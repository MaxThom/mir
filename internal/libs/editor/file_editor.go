package editor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

func EditRawDocument(data *[]byte, headers []string) error {
	ext := "json"
	comment := "// "

	for i := range headers {
		headers[i] = comment + headers[i] + "\n"
	}
	bData := append([]byte(strings.Join(headers, "")), *data...)

	content, err := interactiveDocumentEdit(bData, "mir_*."+ext)
	if err != nil {
		return fmt.Errorf("can't create or edit temporary file to edit resources: %w", err)
	}
	*data = content

	return nil
}

func EditJsonDocument[T any](data T, headers []string) error {
	ext := "json"
	comment := "// "
	bData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal data: %w", err)
	}

	for i := range headers {
		headers[i] = comment + headers[i] + "\n"
	}
	bData = append([]byte(strings.Join(headers, "")), bData...)

	content, err := interactiveDocumentEdit(bData, "mir_*."+ext)
	if err != nil {
		return fmt.Errorf("can't create or edit temporary file to edit resources: %w", err)
	}

	if err = json.Unmarshal(content, data); err != nil {
		return fmt.Errorf("can't unmarshal data: %w", err)
	}

	return nil
}

func EditYamlDocument[T any](data T, headers []string) error {
	ext := "yaml"
	comment := "# "
	bData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("can't marshal data: %w", err)
	}

	for i := range headers {
		headers[i] = comment + headers[i] + "\n"
	}
	bData = append([]byte(strings.Join(headers, "")), bData...)

	content, err := interactiveDocumentEdit(bData, "mir_*."+ext)
	if err != nil {
		return fmt.Errorf("can't create or edit temporary file to edit resources: %w", err)
	}

	if err = yaml.Unmarshal(content, data); err != nil {
		return fmt.Errorf("can't unmarshal data: %w", err)
	}

	return nil
}

func interactiveDocumentEdit(data []byte, fileName string) ([]byte, error) {
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
	tmpFile, err := os.CreateTemp("", fileName)
	if err != nil {
		return nil, err
	}

	_, err = tmpFile.Write(data)
	if err != nil {
		return nil, err
	}
	tmpFile.Close()

	c := exec.Command(editor, tmpFile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err = c.Run(); err != nil {
		return nil, err
	}

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "//") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			sb.WriteString(line + "\n")
		}
	}
	content := []byte(sb.String())

	err = os.Remove(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	return content, nil
}
