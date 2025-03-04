# autoCommit

Application designed as a `prepare-commit-msg` git hook that analyzes your staged files and generates a commit message **without AI bullshit**!

## Installation

```bash
  go install github.com/burgr033/autoCommit@latest
```

## Configuration

create a `prepare-commit-msg` file in your git repository under `.git/hooks/` directory and put in the following (as an example):

```
#!/bin/sh
COMMIT_MSG_FILE=$1
CUSTOM_HEADER="HERE IS MY CUSTOM HEADER"
CUSTOM_FOOTER="HERE IS MY CUSTOM FOOTER"
autoCommit $COMMIT_MSG_FILE $CUSTOM_HEADER $CUSTOM_FOOTER
```
> [!NOTE]
> this could also be done in a git template. See https://git-scm.com/docs/git-init#_template_directory


## Features

- Pattern matching for file names for special files like `.gitlab-ci.yml` or `docs/*`
- Descriptive comments, depending on git operation
- if pattern matching for files does not bring up a result the branch name is checked for a possible "git flow" naming.

## Examples

The repo with the following `git status` on a branch called `feature/loremipsum`:

```bash
A  .gitlab-ci.yml
A  README.MD
MM internal/filetypes/filetypes.go
MM main.go
?? COMMIT_EDITMSG
?? test.txt
```

generates the following commit message:

```bash
# This is an automated commit message

# docs: added README.MD
# unknown: modified internal/filetypes/filetypes.go, main.go
# ci: added .gitlab-ci.yml

# This is the Footer of the automated commit message
```
