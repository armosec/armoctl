package incidents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const changeStatusPath = "/runtime/incidents/changeStatus"

// validStatuses maps lowercased input to its canonical form.
var validStatuses = map[string]string{
	"open":          "Open",
	"investigating": "Investigating",
	"dismissed":     "Dismissed",
	"resolved":      "Resolved",
}

// normalizeStatus validates and canonicalizes a status string.
func normalizeStatus(s string) (string, error) {
	canon, ok := validStatuses[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return "", &clierr.Error{
			Code: clierr.CodeBadInput,
			Msg:  fmt.Sprintf("invalid --status %q (valid: Open, Investigating, Dismissed, Resolved)", s),
		}
	}
	return canon, nil
}

type statusChangeOpts struct {
	status        string
	guids         []string
	filters       []map[string]string
	searchText    string
	falsePositive bool
	commandName   string
}

// runStatusChange builds and (via safety.Wrap) executes a changeStatus request.
func runStatusChange(cmd *cobra.Command, cli *apiclient.Client, o statusChangeOpts) error {
	if len(o.guids) == 0 && len(o.filters) == 0 && o.searchText == "" {
		return &clierr.Error{
			Code: clierr.CodeBadInput,
			Msg:  "no incidents selected: provide GUID(s), --filter, or --search",
		}
	}

	guids := o.guids
	if guids == nil {
		guids = []string{}
	}
	filters := o.filters
	if filters == nil {
		filters = []map[string]string{}
	}
	body := map[string]any{
		"status":                o.status,
		"incidentsGuids":        guids,
		"innerFilters":          filters,
		"markedAsFalsePositive": o.falsePositive,
	}

	var query url.Values
	preview := map[string]any{"method": "POST", "url": changeStatusPath, "body": body}
	if o.searchText != "" {
		query = url.Values{"searchText": {o.searchText}}
		preview["query"] = map[string]any{"searchText": o.searchText}
	}

	m := cliflags.ReadMutation(cmd)

	return safety.Wrap(cmd.Context(), safety.Args{
		Command: o.commandName,
		DryRun:  m.DryRun,
		Yes:     m.Yes,
		Tty:     term.IsTerminal(int(os.Stdin.Fd())),
		Stdout:  cmd.OutOrStdout(),
		Stderr:  cmd.ErrOrStderr(),
		Preview: preview,
		ArgsLog: statusChangeArgsLog(o),
		Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
			resp, err := cli.Do(ctx, http.MethodPost, changeStatusPath, query, body)
			if err != nil {
				return nil, safety.ExecMeta{}, err
			}
			defer func() { _ = resp.Body.Close() }()
			raw, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				return nil, safety.ExecMeta{}, &clierr.Error{
					Code:      clierr.CodeServer,
					Msg:       strings.TrimSpace(string(raw)),
					RequestID: resp.Header.Get("x-request-id"),
				}
			}
			var out map[string]any
			_ = json.Unmarshal(raw, &out)
			return out, safety.ExecMeta{
				URL:       "POST " + changeStatusPath,
				Status:    resp.StatusCode,
				RequestID: resp.Header.Get("x-request-id"),
			}, nil
		},
	})
}

// statusChangeArgsLog renders a one-line audit summary of the selection.
func statusChangeArgsLog(o statusChangeOpts) string {
	parts := []string{"status=" + o.status}
	if len(o.guids) > 0 {
		parts = append(parts, fmt.Sprintf("guids=%d", len(o.guids)))
	}
	if len(o.filters) > 0 {
		parts = append(parts, fmt.Sprintf("filters=%v", o.filters))
	}
	if o.searchText != "" {
		parts = append(parts, "search="+o.searchText)
	}
	if o.falsePositive {
		parts = append(parts, "falsePositive=true")
	}
	return strings.Join(parts, " ")
}
