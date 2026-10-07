package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type PageOptions struct {
	Page     int
	PageSize int
	State    string
	Search   string
	Focus    string
}

type Page struct {
	Items      []Job `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int   `json:"total"`
	TotalPages int   `json:"total_pages"`
}

var actionLabels = map[string]string{
	"vm.interface.add":    "Add VM interface",
	"vm.interface.update": "Update VM interface",
	"vm.interface.remove": "Remove VM interface",
	"vm.create":           "Create virtual machine",
	"vm.start":            "Start virtual machine",
	"vm.shutdown":         "Guest shutdown",
	"vm.reboot":           "Reboot virtual machine",
	"vm.stop":             "Stop virtual machine",
	"vm.delete":           "Delete virtual machine",
	"vm.hardware":         "Update VM hardware",
	"vm.guest-setup":      "Update guest setup",
	"vm.disk.add":         "Add VM disk",
	"vm.disk.grow":        "Grow VM disk",
	"vm.disk.remove":      "Remove VM disk",
	"vm.disk.replace":     "Replace VM boot disk",
	"vm.disk.wipe":        "Wipe VM disk",
	"vm.backup":           "Back up virtual machine",
	"vm.backup.restore":   "Restore VM backup",
	"vm.backup.delete":    "Delete VM backup",
	"vm.backup.prune":     "Prune VM backups",
	"vm.snapshot.create":  "Create VM snapshot",
	"vm.snapshot.restore": "Restore VM snapshot",
	"vm.snapshot.delete":  "Delete VM snapshot",
	"network.update":      "Edit network",
	"network.create":      "Create network",
	"network.apply":       "Apply network",
	"network.destroy":     "Delete network",
	"boot.reconcile":      "Reconcile on boot",
	"host.poweroff":       "Power off host",
	"host.reboot":         "Reboot host",
}

func Label(action string) string {
	if label, ok := actionLabels[action]; ok {
		return label
	}

	return action
}

func matchesPage(job Job, options PageOptions) bool {
	if job.Private() {
		return false
	}

	switch options.State {
	case "active":
		if job.State != Pending && job.State != Running {
			return false
		}
	case "completed":
		if job.State != Succeeded && job.State != Failed {
			return false
		}
	case "", "all":
	default:
		if string(job.State) != options.State {
			return false
		}
	}

	text := strings.ToLower(job.Action + " " + actionLabels[job.Action] + " " + job.Target + " " + job.ID)
	return strings.Contains(text, strings.ToLower(strings.TrimSpace(options.Search)))
}

func stateRank(state State) int {
	switch state {
	case Running:
		return 0
	case Pending:
		return 1
	default:
		return 2
	}
}

func (s *Service) PublicPage(ctx context.Context, options PageOptions) (*Page, error) {
	if options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, fmt.Errorf("invalid pagination options")
	}

	ids, err := s.redis.ZRevRange(ctx, s.prefix+"index", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	matches := make([]Job, 0)
	for offset := 0; offset < len(ids); offset += 200 {
		end := offset + 200
		if end > len(ids) {
			end = len(ids)
		}

		keys := make([]string, 0, end-offset)
		for _, id := range ids[offset:end] {
			keys = append(keys, s.prefix+id)
		}

		values, err := s.redis.MGet(ctx, keys...).Result()
		if err != nil {
			return nil, err
		}

		for _, value := range values {
			data, ok := value.(string)
			if !ok {
				continue
			}

			var job Job
			if err := json.Unmarshal([]byte(data), &job); err != nil {
				return nil, err
			}

			if job.State == Pending || job.State == Running {
				if refreshed, err := s.Get(ctx, job.ID); err == nil {
					job = *refreshed
				}
			}

			job.Label = Label(job.Action)

			if !matchesPage(job, options) {
				continue
			}

			job.Logs = []string{}
			matches = append(matches, job)
		}
	}

	sort.Slice(matches, func(first, second int) bool {
		a, b := matches[first], matches[second]
		if a.ID == options.Focus || b.ID == options.Focus {
			return a.ID == options.Focus
		}

		if stateRank(a.State) != stateRank(b.State) {
			return stateRank(a.State) < stateRank(b.State)
		}

		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}

		return a.ID < b.ID
	})
	totalPages := (len(matches) + options.PageSize - 1) / options.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	page := options.Page
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * options.PageSize
	end := start + options.PageSize
	if end > len(matches) {
		end = len(matches)
	}

	return &Page{Items: matches[start:end], Page: page, PageSize: options.PageSize, Total: len(matches), TotalPages: totalPages}, nil
}
