package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const goSkillPrompt = `# Let's Go: Compass Task Runner

Execute the following steps in order.

## Step 1: Learn Compass CLI

Run ` + "`compass --mtp-describe`" + ` and read the output to understand all available Compass commands and usage patterns. Do not skip this step.

## Step 2: Find the Next Ready Task

Run ` + "`compass task ready`" + ` to list tasks that are ready to be worked on. Present the list to the user and pick the top task unless the user specifies otherwise.

## Step 3: Claim the Task

Run ` + "`compass task start <id>`" + ` using the selected task's ID to mark it as in-progress and assign it to yourself.

## Step 4: Understand the Task

Read the task description carefully. It should contain all the context needed: code references, file paths, acceptance criteria, links to related work. If the task description is insufficient, investigate the codebase to fill in gaps before writing code.

## Step 5: Do the Work

Implement the task. Follow the project's conventions, patterns, and quality standards. Refer to CLAUDE.md for project-specific guidance. Run relevant quality gates (tests, linting, typechecking) as you go to verify correctness.

## Step 6: Verify Completion

Before closing the task, confirm ALL of the following:

1. **All acceptance criteria met** per the task description
2. **Quality gates pass**: run the project's test suite, linter, and type checker as appropriate
3. **Changes are committed** with a conventional commit message
4. **No regressions introduced**: existing tests still pass

If any verification fails, fix the issue before proceeding.

## Step 7: Close the Task and Stop

Run ` + "`compass task close <id>`" + ` to mark the task as complete. Then stop. Do not pick up another task.
`

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "Print the lets-go skill prompt for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(goSkillPrompt)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
}
