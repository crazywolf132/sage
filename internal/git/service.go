package git

type Service interface {
	IsRepo() (bool, error)
	IsClean() (bool, error)
	StageAll() error
	Commit(msg string, allowEmpty bool) error
	CurrentBranch() (string, error)
	Push(branch string, force bool) error
	GetDiff() (string, error)
	DefaultBranch() (string, error)
	MergedBranches(base string) ([]string, error)
	DeleteBranch(name string) error
	FetchAll() error
	Checkout(name string) error
	Pull() error
	CreateBranch(name string) error
	Merge(base string) error
	MergeAbort() error
	IsMerging() (bool, error)
	RebaseAbort() error
	IsRebasing() (bool, error)
	StatusPorcelain() (string, error)
	ResetSoft(ref string) error
	ListBranches() ([]string, error)
	Log(branch string, limit int, stats, all bool) (string, error)
}
