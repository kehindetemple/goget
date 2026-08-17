// Package ui contains small terminal interaction helpers: numbered
// selection menus, text prompts and yes/no confirmations.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// SelectOption prints a 1-indexed numbered menu of labels and returns the
// zero-based index the user chose. Returns (-1, nil) if the user cancels
// by entering "0" or an empty line.
func SelectOption(prompt string, labels []string) (int, error) {
	fmt.Println(prompt)
	fmt.Println()
	for i, l := range labels {
		fmt.Printf("  %d. %s\n", i+1, l)
	}
	fmt.Println()
	fmt.Print("Select (or 0 to cancel): ")

	line, err := reader.ReadString('\n')
	if err != nil {
		return -1, err
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "0" {
		return -1, nil
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(labels) {
		return -1, fmt.Errorf("invalid selection %q", line)
	}
	return n - 1, nil
}

// Confirm asks a yes/no question, defaulting to "no" on empty input.
func Confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// Text prompts for a single line of free text input.
func Text(prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
