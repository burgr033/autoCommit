package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/burgr033/autoCommit/internal/filetypes"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// custom FOOTER and HEADER command line flags
var (
	HEADER = ""
	FOOTER = ""
)

type (
	CommitBody    []Msg
	GroupedCommit map[string][]string
	Msg           struct {
		Conventional string
		GitStatus    string
		File         string
		Extra        string
	}
)

// Cache for conventional type lookups
var (
	conventionalTypeCache = make(map[string]string)
	cacheMutex            sync.RWMutex
)

func (b *CommitBody) toString() string {
	var bodyString []string
	if HEADER != "" {
		bodyString = append(bodyString, HEADER)
		bodyString = append(bodyString, "#")
	}
	grouped := b.groupMessages()
	for key, files := range grouped {
		bodyString = append(bodyString, fmt.Sprintf("# %s %s", key, strings.Join(files, ", ")))
	}
	if FOOTER != "" {
		bodyString = append(bodyString, "#")
		bodyString = append(bodyString, FOOTER)
	}
	return strings.Join(bodyString, "\n")
}

func (b *CommitBody) groupMessages() GroupedCommit {
	grouped := make(GroupedCommit)
	for _, msg := range *b {
		key := fmt.Sprintf("%s: %s", msg.Conventional, msg.GitStatus)
		grouped[key] = append(grouped[key], msg.File)
	}
	return grouped
}

func getConventionalType(filename string) string {
	lowerFilename := strings.ToLower(filename)

	cacheMutex.RLock()
	if commitType, exists := conventionalTypeCache[lowerFilename]; exists {
		cacheMutex.RUnlock()
		return commitType
	}
	cacheMutex.RUnlock()

	var commitType string

	if commitType, exists := filetypes.NameMapping[lowerFilename]; exists {
		cacheMutex.Lock()
		conventionalTypeCache[lowerFilename] = commitType
		cacheMutex.Unlock()
		return commitType
	}

	for pattern, cType := range filetypes.NameMapping {
		if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(lowerFilename, dir+"/") {
				commitType = cType
				break
			}
		}
	}

	if commitType == "" {
		base := filepath.Base(lowerFilename)
		for pattern, cType := range filetypes.NameMapping {
			if strings.HasPrefix(pattern, "*") {
				if match, _ := filepath.Match(pattern, base); match {
					commitType = cType
					break
				}
			}
		}
	}

	if commitType == "" {
		commitType = filetypes.ConventionalUnknown
	}

	cacheMutex.Lock()
	conventionalTypeCache[lowerFilename] = commitType
	cacheMutex.Unlock()

	return commitType
}

func getNamingOfBranch(branch string) string {
	branchSplit := strings.Split(branch, "/")
	if len(branchSplit) > 0 {
		if commitType, exists := filetypes.BranchMapping[branchSplit[0]]; exists {
			return commitType
		}
	}
	return filetypes.ConventionalUnknown
}

func getGitStatusText(gs git.StatusCode) string {
	switch gs {
	case git.Modified:
		return "modified"
	case git.Added:
		return "added"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	default:
		return ""
	}
}

func determineGitStatus(repo *git.Repository) CommitBody {
	wt, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Failed to get worktree: %v", err)
	}

	status, err := wt.Status()
	if err != nil {
		log.Fatalf("Failed to get status from worktree: %v", err)
	}
	var head *plumbing.Reference
	head, err = repo.Head()
	if err != nil {
		head, _ = repo.Reference(plumbing.HEAD, false)
	}
	branchName := head.Name().Short()
	branchType := getNamingOfBranch(branchName)

	var messages []Msg
	for file, statusEntry := range status {

		if statusEntry.Staging == git.Untracked {
			continue
		}

		gitStatusText := getGitStatusText(statusEntry.Staging)
		if gitStatusText == "" {
			continue
		}

		var message Msg
		message.File = file
		message.GitStatus = gitStatusText

		message.Conventional = getConventionalType(file)
		if message.Conventional == filetypes.ConventionalUnknown {
			message.Conventional = branchType
		}

		messages = append(messages, message)
	}
	return messages
}

func main() {
	var toStdout bool

	flag.BoolVar(&toStdout, "stdout", false, "write to stdout and don't overwrite a file.")
	flag.Parse()

	args := flag.Args()

	var commitMsgFile, HEADER, FOOTER string

	if len(args) >= 1 {
		commitMsgFile = args[0]
	} else {
		log.Printf("Commit message file not provided. Defaulting to stdout.\n\n")
		toStdout = true
	}

	if len(args) >= 2 {
		HEADER = args[1]
	}

	if len(args) >= 3 {
		FOOTER = args[2]
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatalf("Not in a git repository: %v", err)
	}

	statusString := determineGitStatus(repo)

	// If HEADER or FOOTER is used, prepend/append them to the message
	message := fmt.Sprintf("%s\n%s\n%s", HEADER, statusString.toString(), FOOTER)

	if toStdout {
		fmt.Printf("%s", message)
		return
	}

	err = os.WriteFile(commitMsgFile, []byte(message), 0o644)
	if err != nil {
		log.Fatalf("Error writing to commit message file: %v", err)
	}
}
