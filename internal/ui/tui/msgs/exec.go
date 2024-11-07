package msgs

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
)

type (
	EditorFinishedMsg struct {
		Content json.RawMessage
		Err     error
	}
)

func EditorFinishedCmd(content json.RawMessage, err error) tea.Cmd {
	return func() tea.Msg {
		return EditorFinishedMsg{Content: content, Err: err}
	}
}

// headerComments are comments that will be added to the top of the file
// with // or # for comments. Can be used to include instruction to the file
func OpenEditorCmd(data json.RawMessage, headerComments []string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		switch runtime.GOOS {
		case "windows":
			editor = "notepad.exe"
		case "linux":
			editor = "nano"
		case "darwin":
			editor = "nano"
		default:
			editor = "nano"
		}
	}

	// Create a temporary file
	// TODO option for user to switch between json and yaml
	tmpfile, err := os.CreateTemp("", "twin_*.json")
	if err != nil {
		return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't create temporary file for editing twin"))
	}

	// Write initial device to the temp file
	for i := range headerComments {
		headerComments[i] = "// " + headerComments[i] + "\n"
	}

	_, err = tmpfile.Write([]byte(strings.Join(headerComments, "")))
	if err != nil {
		return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't write to temporary file for editing twin"))
	}
	_, err = tmpfile.Write(data)
	if err != nil {
		return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't write to temporary file for editing twin"))
	}
	tmpfile.Close()

	c := exec.Command(editor, tmpfile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't run editor command for editing twin"))
		}

		file, err := os.Open(tmpfile.Name())
		if err != nil {
			return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't open to temporary file for reading twin"))
		}

		var sb strings.Builder
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				sb.WriteString(line)
			}
		}
		content := []byte(sb.String())

		err = os.Remove(tmpfile.Name())
		if err != nil {
			return EditorFinishedCmd([]byte{}, errors.Wrap(err, "can't delete temporary file for editing twin"))
		}

		return EditorFinishedMsg{Content: content, Err: nil}
	})
}
