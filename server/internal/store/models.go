package store

// Rows travel as JSON with camelCase names, the same in pull, push and
// changes. Nullable columns are pointers so that JSON shows null.

type Goal struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Notes      string  `json:"notes"`
	Status     string  `json:"status"`
	TargetDate *string `json:"targetDate"`
	SortKey    float64 `json:"sortKey"`
	CreatedAt  int64   `json:"createdAt"`
	UpdatedAt  int64   `json:"updatedAt"`
	DeletedAt  *int64  `json:"deletedAt"`
	Rev        int64   `json:"rev"`
}

type Project struct {
	ID        string  `json:"id"`
	GoalID    *string `json:"goalId"`
	Title     string  `json:"title"`
	Notes     string  `json:"notes"`
	Status    string  `json:"status"`
	Due       *string `json:"due"`
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
	DeletedAt *int64  `json:"deletedAt"`
	Rev       int64   `json:"rev"`
}

type Repeat struct {
	ID        string  `json:"id"`
	Freq      string  `json:"freq"`
	Every     int     `json:"every"`
	Weekdays  *int    `json:"weekdays"`
	Mode      string  `json:"mode"`
	Until     *string `json:"until"`
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
	DeletedAt *int64  `json:"deletedAt"`
	Rev       int64   `json:"rev"`
}

type Task struct {
	ID        string   `json:"id"`
	ProjectID *string  `json:"projectId"`
	GoalID    *string  `json:"goalId"`
	Title     string   `json:"title"`
	Notes     string   `json:"notes"`
	Status    string   `json:"status"`
	Due       *string  `json:"due"`
	PlannedOn *string  `json:"plannedOn"`
	PlanRank  *float64 `json:"planRank"`
	Slipped   int      `json:"slipped"`
	RepeatID  *string  `json:"repeatId"`
	DoneAt    *int64   `json:"doneAt"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
	DeletedAt *int64   `json:"deletedAt"`
	Rev       int64    `json:"rev"`
}

type Attachment struct {
	ID        string  `json:"id"`
	GoalID    *string `json:"goalId"`
	ProjectID *string `json:"projectId"`
	TaskID    *string `json:"taskId"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	URL       *string `json:"url"`
	Body      *string `json:"body"`
	FileSHA   *string `json:"fileSha"`
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
	DeletedAt *int64  `json:"deletedAt"`
	Rev       int64   `json:"rev"`
}

type Reminder struct {
	Slot      int    `json:"slot"`
	Enabled   bool   `json:"enabled"`
	AtLocal   string `json:"atLocal"`
	Kind      string `json:"kind"`
	UpdatedAt int64  `json:"updatedAt"`
	Rev       int64  `json:"rev"`
}

// File is a row of the files table; it is not synced.
type File struct {
	SHA       string
	Mime      string
	Size      int64
	CreatedAt int64
}

type Settings struct {
	TZ            string
	WeekStart     int
	CalendarToken string
	LastRollover  *string
	Backups       bool
}

// PushSub is a browser's Web Push subscription; it is not synced.
type PushSub struct {
	Endpoint  string
	P256DH    string
	Auth      string
	Label     string
	CreatedAt int64
	LastOK    *int64
}

// Account is this database's person. ChangedAt is when the passphrase was
// set; Owner may add and remove people; MustChange is set for an added
// person until they choose their own passphrase.
type Account struct {
	Username       string
	PassphraseHash string
	ChangedAt      int64
	Owner          bool
	MustChange     bool
}

type Session struct {
	TokenHash string
	Label     string
	CreatedAt int64
	LastSeen  int64
}

// Changes is the set of synced rows returned by a pull or a push. Slices are
// never nil so that JSON always shows arrays.
type Changes struct {
	Goals       []Goal       `json:"goals"`
	Projects    []Project    `json:"projects"`
	Tasks       []Task       `json:"tasks"`
	Repeats     []Repeat     `json:"repeats"`
	Attachments []Attachment `json:"attachments"`
	Reminders   []Reminder   `json:"reminders"`
}

// EmptyChanges returns Changes with empty, non-nil slices.
func EmptyChanges() Changes {
	return Changes{
		Goals:       []Goal{},
		Projects:    []Project{},
		Tasks:       []Task{},
		Repeats:     []Repeat{},
		Attachments: []Attachment{},
		Reminders:   []Reminder{},
	}
}

// OwnerKind names the table an attachment belongs to.
type OwnerKind string

const (
	OwnerGoal    OwnerKind = "goal"
	OwnerProject OwnerKind = "project"
	OwnerTask    OwnerKind = "task"
)
