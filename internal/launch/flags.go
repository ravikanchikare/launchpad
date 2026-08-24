package launch

import (
	"errors"
	"strings"
)

func ParseModelFlag(args []string) (string, []string, error) {
	for i, arg := range args {
		if arg == "--model" {
			if i+1 >= len(args) || args[i+1] == "" {
				return "", nil, errors.New("--model requires a value")
			}
			return args[i+1], args, nil
		}
		if strings.HasPrefix(arg, "--model=") {
			model := strings.TrimPrefix(arg, "--model=")
			if model == "" {
				return "", nil, errors.New("--model requires a value")
			}
			return model, args, nil
		}
	}
	return "", args, nil
}

func StripModelFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--model" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--model=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func HasRestoreFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--restore" {
			return true
		}
	}
	return false
}

func StripRestoreFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--restore" {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func ParseKeyFlag(args []string) (string, []string, error) {
	for i, arg := range args {
		if arg == "--key" {
			if i+1 >= len(args) || args[i+1] == "" {
				return "", nil, errors.New("--key requires a value")
			}
			remaining := append([]string{}, args[:i]...)
			return args[i+1], append(remaining, args[i+2:]...), nil
		}
		if strings.HasPrefix(arg, "--key=") {
			key := strings.TrimPrefix(arg, "--key=")
			if key == "" {
				return "", nil, errors.New("--key requires a value")
			}
			return key, append(append([]string{}, args[:i]...), args[i+1:]...), nil
		}
	}
	return "", args, nil
}

func StripKeyFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--key" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--key=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func EnsureModelArg(args []string, model string) []string {
	if model == "" {
		return args
	}
	selected, _, err := ParseModelFlag(args)
	if err == nil && selected != "" {
		return args
	}
	return append([]string{"--model", model}, args...)
}
