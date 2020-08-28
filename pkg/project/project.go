package project

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/observatorium/statectl/pkg/merrors"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sourcegraph/go-diff/diff"
)

// Ref represents git repository version that can be checked out. In practice this means either
// commit SHA or git tag.
type Ref string

func (r *Ref) Commit() plumbing.Hash {
	isHash, err := regexp.MatchString("^[0-9a-z]{64}&", string(*r))
	if err != nil {
		panic(err) // Panic only if pattern is not compilable.
	}
	// TODO(bwplotka): Actually support tags.
	if isHash {
		return plumbing.NewHash(string(*r))
	}
	// Assume branch.
	return plumbing.NewReferenceFromStrings("origin", string(*r)).Hash()
}

type Project struct {
	ctx context.Context

	Logger log.Logger

	cacheDir string
	cfg      *Config

	// TODO(bwplotka): Introduce interfaces & add unit tests.
	state         *repo
	configuration *repo
}

type repo struct {
	wt   *git.Worktree
	repo *git.Repository
}

func (r *repo) fetch(ctx context.Context) error {
	if err := r.repo.FetchContext(ctx, &git.FetchOptions{
		Tags:  git.AllTags,
		Force: true,
	}); err != nil && errors.Cause(err) != git.NoErrAlreadyUpToDate {
		return errors.Wrap(err, "fetch")
	}
	return nil
}

func (r *repo) reset(ref Ref) error {
	if err := r.wt.Reset(&git.ResetOptions{
		Commit: ref.Commit(),
		Mode:   git.HardReset,
	}); err != nil {
		return errors.Wrap(err, "reset")
	}
	return nil
}

func New(ctx context.Context, logger log.Logger, cfg *Config, cacheDir string) (*Project, error) {
	p := &Project{
		ctx:      ctx,
		Logger:   logger,
		cacheDir: cacheDir,
		cfg:      cfg,
	}
	level.Debug(logger).Log("msg", "opening state repo", "url", cfg.State.URL, "dir", p.stateLocalRepoDir())
	stateRepo, err := git.PlainOpen(p.stateLocalRepoDir())
	if errors.Cause(err) == git.ErrRepositoryNotExists {
		level.Debug(logger).Log("msg", "state repo not found, re-cloning", "url", cfg.State.URL, "dir", p.stateLocalRepoDir())
		stateRepo, err = git.PlainCloneContext(ctx, p.stateLocalRepoDir(), false, &git.CloneOptions{
			URL:        cfg.State.URL,
			NoCheckout: true,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "clone state repo %v to %v", cfg.State.URL, p.stateLocalRepoDir())
		}
	} else {
		if err != nil {
			return nil, errors.Wrapf(err, "open state repo %v to %v", cfg.State.URL, p.stateLocalRepoDir())
		}
	}

	p.state = &repo{repo: stateRepo}
	p.state.wt, err = stateRepo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "state worktree")
	}

	level.Debug(logger).Log("msg", "opening configuration repo", "url", cfg.Configuration.URL, "dir", p.configurationLocalRepoDir())
	configurationRepo, err := git.PlainOpen(p.configurationLocalRepoDir())
	if errors.Cause(err) == git.ErrRepositoryNotExists {
		level.Debug(logger).Log("msg", "configuration repo not found, re-cloning", "url", cfg.Configuration.URL, "dir", p.configurationLocalRepoDir())
		configurationRepo, err = git.PlainClone(p.configurationLocalRepoDir(), false, &git.CloneOptions{
			URL:        cfg.Configuration.URL,
			NoCheckout: true,
		})
	}
	if err != nil {
		return nil, errors.Wrapf(err, "open/clone configuration repo %v to %v", cfg.State.URL, p.configurationLocalRepoDir())
	}
	p.configuration = &repo{repo: configurationRepo}
	p.configuration.wt, err = configurationRepo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "configuration worktree")
	}
	return p, nil
}

func (p *Project) stateLocalRepoDir() string {
	return filepath.Join(p.cacheDir, "state", p.cfg.State.URL)
}

func (p *Project) configurationLocalRepoDir() string {
	return filepath.Join(p.cacheDir, "configuration", p.cfg.Configuration.URL)
}

func (p *Project) GitDiffConfig(w io.Writer, base, new Ref) error {
	// TODO(bwplotka): Would be nice to show com log as well on top of state diff e.g
	// e.g p.configuration.repo.Log(&git.LogOptions{From: base.Commit()})
	return errors.New("not implemented")
}

func (p *Project) DiffState(w io.Writer, base, new Ref) error {
	logger := log.With(p.Logger, "baseRef", base, "newRef", new)
	level.Debug(logger).Log("msg", "comparing configuration states")
	if err := p.state.fetch(p.ctx); err != nil {
		return err
	}
	if err := p.configuration.fetch(p.ctx); err != nil {
		return err
	}

	if err := p.state.reset(base); err != nil {
		return err
	}
	dplStatesBase, err := p.cfg.State.Codec.Decode(p.stateLocalRepoDir())
	if err != nil {
		return err
	}
	level.Debug(logger).Log("msg", "fetched base states", "deployments", len(dplStatesBase))
	if err := p.state.reset(new); err != nil {
		return err
	}
	dplStatesNew, err := p.cfg.State.Codec.Decode(p.stateLocalRepoDir())
	if err != nil {
		return err
	}
	level.Debug(logger).Log("msg", "fetched new states", "deployments", len(dplStatesNew))

	getDeploymentFn := func(state ServiceState, ref Ref) (string, error) {
		k := dplKey{service: state.Service, cluster: state.Cluster}
		// TODO(bwplotka): Assume same repo.
		if err := p.configuration.reset(state.ConfigurationRef); err != nil {
			return "", errors.Wrapf(err, "checkout configuration referenced as %q by state %v for %v", state.ConfigurationRef, ref, k)
		}
		b, err := ioutil.ReadFile(filepath.Join(p.configurationLocalRepoDir(), state.ConfigurationPath))
		if err != nil {
			return "", errors.Wrapf(err, "invalid state; read configuration file referenced as %q by state %v for %v", state.ConfigurationRef, ref, k)
		}

		content := string(b)
		for k, v := range state.EnvParameters {
			content = strings.ReplaceAll(content, fmt.Sprintf("${%s}", k), v)
		}
		return content, nil
	}

	diffErr := merrors.New()
	baseDeployments := map[dplKey]string{}
	baseDeploymentsErrored := map[dplKey]struct{}{}
	for _, state := range dplStatesBase {
		content, err := getDeploymentFn(state, base)
		k := dplKey{service: state.Service, cluster: state.Cluster}
		if err != nil {
			baseDeploymentsErrored[k] = struct{}{}
			diffErr.Add(err)
			continue
		}
		baseDeployments[k] = content
	}

	visited := map[dplKey]struct{}{}
	for _, state := range dplStatesNew {
		k := dplKey{service: state.Service, cluster: state.Cluster}
		content, err := getDeploymentFn(state, new)
		if err != nil {
			visited[k] = struct{}{}
			diffErr.Add(err)
			continue
		}

		base, ok := baseDeployments[k]
		if ok {
			visited[k] = struct{}{}
		} else {
			if _, ok := baseDeploymentsErrored[k]; ok {
				visited[k] = struct{}{}
				continue
			}
		}

		// Never seen before deployment are just empty string, so proper diff will be printed.
		printDiff(w, k, base, content)
	}

	for k, base := range baseDeployments {
		if _, ok := visited[k]; ok {
			continue
		}
		// Deployment not present in new state.
		printDiff(w, k, base, "")
	}

	return diffErr.Err()
}

func printDiff(w io.Writer, key dplKey, base, new string) {
	_, _ = w.Write([]byte(fmt.Sprintf("Service: %q Deploying to %v \n", key.service, key.cluster)))
	dmp := diffmatchpatch.New()

	//diffs := dmp.DiffMain(base, new, true)
	//diffs = dmp.DiffCleanupSemantic(diffs)
	//diffs = dmp.DiffCleanupEfficiency(diffs)
	//
	//for _, d := range diffs {
	//	if d.Type != diffmatchpatch.DiffEqual {
	//		continue
	//	}
	//}
	//_, _ = w.Write([]byte(dmp.DiffPrettyText(diffs)))
	f, err := diff.ParseFileDiff([]byte(dmp.PatchToText(dmp.PatchMake(base, new))))
	if err != nil {
		panic(err)
	}

	b, err := diff.PrintFileDiff(f)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))
}

type dplKey struct {
	cluster Cluster
	service string
}

type StateCodec interface {
	Decode(dir string) ([]ServiceState, error)
	Encode(dir string, s []ServiceState) error
}

type ServiceState struct {
	Service string
	Cluster Cluster

	ConfigurationRef  Ref
	ConfigurationURL  string
	ConfigurationPath string
	EnvParameters     map[string]string
}
