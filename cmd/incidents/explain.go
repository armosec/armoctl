package incidents

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExplainCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "explain [guid]",
		Short: "Aggregate the platform's streaming explanation for an incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "explain requires a GUID"}
			}
			cli := clientFor(cmd)
			path := "/runtime/incidents/" + args[0] + "/explain"
			resp, err := cli.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return &clierr.Error{Code: codeForStatus(resp.StatusCode), Msg: strings.TrimSpace(string(b))}
			}
			var sb strings.Builder
			sc := bufio.NewScanner(resp.Body)
			sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
			for sc.Scan() {
				line := sc.Text()
				if !strings.HasPrefix(line, "data: ") {
					continue
				}
				payload := strings.TrimPrefix(line, "data: ")
				if payload == "[DONE]" {
					break
				}
				if payload == "[CACHED]" {
					continue
				}
				var chunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
					continue // skip malformed
				}
				if len(chunk.Choices) > 0 {
					sb.WriteString(chunk.Choices[0].Delta.Content)
				}
			}
			if err := sc.Err(); err != nil {
				return err
			}
			obj := map[string]any{"incidentGUID": args[0], "explanation": sb.String()}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}

// codeForStatus returns the appropriate clierr.Code for an HTTP status.
func codeForStatus(s int) clierr.Code {
	switch {
	case s == 401, s == 403:
		return clierr.CodeAuth
	case s == 404:
		return clierr.CodeNotFound
	case s >= 400 && s < 500:
		return clierr.CodeBadInput
	default:
		return clierr.CodeServer
	}
}
