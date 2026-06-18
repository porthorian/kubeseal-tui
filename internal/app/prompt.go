package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Prompt struct {
	in  io.Reader
	out io.Writer
	err io.Writer
	cwd string
	r   *bufio.Reader
}

func NewPrompt(in io.Reader, out io.Writer, errOut io.Writer, cwd string) *Prompt {
	return &Prompt{
		in:  in,
		out: out,
		err: errOut,
		cwd: cwd,
		r:   bufio.NewReader(in),
	}
}

func (p *Prompt) Run(cfg *Config) error {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return errors.New("interactive mode requires a TTY. Use non-interactive flags instead")
	}

	if err := p.promptSecretName(cfg); err != nil {
		return err
	}
	if err := p.promptSecretType(cfg); err != nil {
		return err
	}
	if err := p.collectData(cfg); err != nil {
		return err
	}
	if err := p.collectTargets(cfg); err != nil {
		return err
	}
	if err := ValidateAndPrepare(cfg); err != nil {
		return err
	}
	p.printReviewSummary(*cfg)
	if !p.confirm("Proceed with sealing and writing files?", false) {
		return errors.New("aborted by user")
	}
	for _, path := range ExistingOutputFiles(cfg.Targets) {
		if !p.confirm(fmt.Sprintf("Output exists (%s). Overwrite?", path), false) {
			return fmt.Errorf("aborted because overwrite was not confirmed for: %s", path)
		}
	}
	return nil
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (p *Prompt) promptSecretName(cfg *Config) error {
	for {
		prompt := "Secret name"
		if TrimSpace(cfg.SecretName) != "" {
			prompt += " [" + cfg.SecretName + "]"
		}
		input, err := p.readLine(prompt + ": ")
		if err != nil {
			return err
		}
		input = TrimSpace(input)
		if input == "" {
			input = TrimSpace(cfg.SecretName)
		}
		if input != "" {
			cfg.SecretName = input
			return nil
		}
		fmt.Fprintln(p.out, "Secret name cannot be empty.")
	}
}

func (p *Prompt) promptSecretType(cfg *Config) error {
	current := TrimSpace(cfg.SecretType)
	if current == "" {
		current = "Opaque"
	}
	options := []string{
		"Opaque",
		"kubernetes.io/basic-auth",
		"kubernetes.io/dockerconfigjson",
		"kubernetes.io/tls",
		"Custom value",
	}
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "Secret type")
	for i, option := range options {
		fmt.Fprintf(p.out, "  %d) %s\n", i+1, option)
	}
	for {
		input, err := p.readLine(fmt.Sprintf("Choose secret type [1, current: %s]: ", current))
		if err != nil {
			return err
		}
		input = TrimSpace(input)
		if input == "" {
			cfg.SecretType = current
			return nil
		}
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(options) {
			fmt.Fprintln(p.out, "Please choose a listed number.")
			continue
		}
		selected := options[choice-1]
		if selected != "Custom value" {
			cfg.SecretType = selected
			return nil
		}
		custom, err := p.readLine(fmt.Sprintf("Custom secret type [%s]: ", current))
		if err != nil {
			return err
		}
		custom = TrimSpace(custom)
		if custom == "" {
			custom = current
		}
		cfg.SecretType = custom
		return nil
	}
}

func (p *Prompt) collectData(cfg *Config) error {
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "Configure secret data entries.")
	for {
		if len(cfg.Data) > 0 && !p.confirm("Add another data entry?", true) {
			return nil
		}

		key, err := p.promptDataKey(cfg)
		if err != nil {
			return err
		}
		mode, err := p.promptInputMode(key)
		if err != nil {
			return err
		}
		switch mode {
		case "multiline":
			value, err := p.readMultilineValue(key)
			if err != nil {
				return err
			}
			if err := AddDataValue(cfg, key, value); err != nil {
				return err
			}
		case "file":
			path, err := p.promptFilePath(key)
			if err != nil {
				return err
			}
			if err := AddDataFile(cfg, key, path); err != nil {
				return err
			}
		default:
			value, err := p.readLine(fmt.Sprintf("Value for %q: ", key))
			if err != nil {
				return err
			}
			if err := AddDataValue(cfg, key, value); err != nil {
				return err
			}
		}
		fmt.Fprintf(p.out, "Added key %q.\n", key)
	}
}

func (p *Prompt) promptDataKey(cfg *Config) (string, error) {
	for {
		key, err := p.readLine("Data key: ")
		if err != nil {
			return "", err
		}
		key = TrimSpace(key)
		if key == "" {
			fmt.Fprintln(p.out, "Key cannot be empty.")
			continue
		}
		if dataKeyExists(cfg.Data, key) {
			fmt.Fprintf(p.out, "Key %q already exists.\n", key)
			continue
		}
		return key, nil
	}
}

func (p *Prompt) promptInputMode(key string) (string, error) {
	for {
		input, err := p.readLine(fmt.Sprintf("Input mode for %q - (s)ingle-line, (m)ultiline, (f)ile [s]: ", key))
		if err != nil {
			return "", err
		}
		input = strings.ToLower(TrimSpace(input))
		if input == "" {
			input = "s"
		}
		switch input {
		case "s", "single":
			return "single", nil
		case "m", "multiline":
			return "multiline", nil
		case "f", "file":
			return "file", nil
		default:
			fmt.Fprintln(p.out, "Please choose s, m, or f.")
		}
	}
}

func (p *Prompt) readMultilineValue(key string) (string, error) {
	const sentinel = "__END__"
	fmt.Fprintf(p.out, "Enter multiline value for %q.\n", key)
	fmt.Fprintf(p.out, "Finish with a line containing only %s.\n", sentinel)
	var lines []string
	for {
		line, err := p.readLine("")
		if err != nil {
			return "", err
		}
		if line == sentinel {
			return strings.Join(lines, "\n"), nil
		}
		lines = append(lines, line)
	}
}

func (p *Prompt) promptFilePath(key string) (string, error) {
	for {
		input, err := p.readLine(fmt.Sprintf("File path for %q: ", key))
		if err != nil {
			return "", err
		}
		path := ExpandHomePath(TrimSpace(input))
		if path == "" {
			fmt.Fprintln(p.out, "File path cannot be empty.")
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			fmt.Fprintf(p.out, "File not readable: %s\n", path)
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(p.out, "File not readable: %s\n", path)
			continue
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		return path, nil
	}
}

func (p *Prompt) collectTargets(cfg *Config) error {
	suggestions := loadNamespaceSuggestions(p.cwd)
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "Configure target namespaces and output directories.")
	for {
		if len(cfg.Targets) > 0 && !p.confirm("Add another target?", true) {
			return nil
		}
		namespace, err := p.promptNamespace(suggestions)
		if err != nil {
			return err
		}
		for {
			input, err := p.readLine(fmt.Sprintf("Output directory for %q (absolute or cwd-relative): ", namespace))
			if err != nil {
				return err
			}
			input = TrimSpace(input)
			if input == "" {
				fmt.Fprintln(p.out, "Directory cannot be empty.")
				continue
			}
			if err := AddTarget(cfg, namespace, input); err != nil {
				fmt.Fprintln(p.out, err)
				continue
			}
			fmt.Fprintf(p.out, "Added target %q -> %q.\n", namespace, cfg.Targets[len(cfg.Targets)-1].Dir)
			break
		}
	}
}

func (p *Prompt) promptNamespace(suggestions []string) (string, error) {
	for {
		if len(suggestions) > 0 {
			fmt.Fprintln(p.out, "Namespace suggestions")
			for i, suggestion := range suggestions {
				fmt.Fprintf(p.out, "  %d) %s\n", i+1, suggestion)
			}
			input, err := p.readLine("Namespace (number or custom): ")
			if err != nil {
				return "", err
			}
			input = TrimSpace(input)
			if choice, err := strconv.Atoi(input); err == nil && choice >= 1 && choice <= len(suggestions) {
				return suggestions[choice-1], nil
			}
			if input != "" {
				return input, nil
			}
		} else {
			input, err := p.readLine("Namespace: ")
			if err != nil {
				return "", err
			}
			input = TrimSpace(input)
			if input != "" {
				return input, nil
			}
		}
		fmt.Fprintln(p.out, "Namespace cannot be empty.")
	}
}

func (p *Prompt) printReviewSummary(cfg Config) {
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "Review")
	fmt.Fprintf(p.out, "  Secret name: %s\n", cfg.SecretName)
	fmt.Fprintf(p.out, "  Secret type: %s\n", cfg.SecretType)
	fmt.Fprintf(p.out, "  Controller: namespace=%s, name=%s\n", cfg.ControllerNamespace, cfg.ControllerName)
	fmt.Fprintf(p.out, "  Data entries: %d\n", len(cfg.Data))
	for _, entry := range cfg.Data {
		source := "inline"
		if entry.Source == DataSourceFile {
			source = "file:" + entry.Payload
		}
		fmt.Fprintf(p.out, "    - %s (%s)\n", entry.Key, source)
	}
	fmt.Fprintf(p.out, "  Targets: %d\n", len(cfg.Targets))
	for _, target := range cfg.Targets {
		fmt.Fprintf(p.out, "    - namespace=%s -> %s\n", target.Namespace, target.File)
	}
}

func (p *Prompt) confirm(prompt string, defaultYes bool) bool {
	for {
		suffix := " [y/N]: "
		defaultAnswer := "n"
		if defaultYes {
			suffix = " [Y/n]: "
			defaultAnswer = "y"
		}
		input, err := p.readLine(prompt + suffix)
		if err != nil {
			return false
		}
		input = strings.ToLower(TrimSpace(input))
		if input == "" {
			input = defaultAnswer
		}
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Fprintln(p.out, "Please answer y or n.")
		}
	}
}

func (p *Prompt) readLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(p.out, prompt)
	}
	line, err := p.r.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && line != "" {
			return strings.TrimRight(line, "\r\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func loadNamespaceSuggestions(cwd string) []string {
	namespaces := map[string]struct{}{}
	if kubectl, err := exec.LookPath("kubectl"); err == nil {
		cmd := exec.Command(kubectl, "get", "ns", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if out, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = TrimSpace(line)
				if line != "" {
					namespaces[line] = struct{}{}
				}
			}
		}
	}
	for _, rel := range []string{"k8/apps", "k8/important"} {
		entries, err := os.ReadDir(filepath.Join(cwd, rel))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				namespaces[entry.Name()] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(namespaces))
	for namespace := range namespaces {
		result = append(result, namespace)
	}
	sort.Strings(result)
	return result
}
