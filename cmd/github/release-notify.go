package github

import (
	"deploy-helper/components/aws/github"
	"fmt"
	"net/http"
	"os"

	gogithub "github.com/google/go-github/github"
	"github.com/nlopes/slack"
	"github.com/spf13/cobra"
)

const noReleaseNotesText = "No release found for this tag. A newer release candidate may have replaced it."

var newGithubClient = func(token string) github.IfaceGithub {
	return github.New(token)
}

func newReleaseNotifyCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "release-notify",
		Short: "get release body and set notify",
		Run:   runReleaseNotifyCmd,
	}

	command.Flags().StringVarP(&inputOwner, "owner", "", "", "github owner(required)")
	command.Flags().StringVarP(&inputRepo, "repo", "", "", "github repo(required)")
	command.Flags().StringVarP(&inputTag, "tag", "", "", "github tag(required)")
	command.Flags().StringVarP(&inputProject, "project", "", "", "project")
	command.Flags().StringVarP(&inputDomain, "domain", "", "", "service domain")
	command.Flags().StringVarP(&inputSlackWebhook, "slackWebhook", "", "", "slack webhook")
	command.Flags().StringVarP(&inputSlackChannel, "slackChannel", "", "", "slack channel")
	command.Flags().BoolVarP(&dryRun, "dry-run", "", false, "dry-run mode for local testing")
	command.MarkFlagRequired("owner")
	command.MarkFlagRequired("repo")
	command.MarkFlagRequired("tag")

	return command
}

func runReleaseNotifyCmd(cmd *cobra.Command, args []string) {
	githubClient := newGithubClient(inputToken)
	repoRelease, err := githubClient.GetReleaseByTag(inputOwner, inputRepo, inputTag)
	releaseURL := repoRelease.GetHTMLURL()
	releaseBody := repoRelease.GetBody()

	if err != nil {
		if !isNotFound(err) {
			panic(err)
		}
		// token 讀不到 private repo 時 GitHub 也回 404；tag 查得到才能確定只是沒有 Release
		if _, refErr := githubClient.GetTagRef(inputOwner, inputRepo, inputTag); refErr != nil {
			panic(fmt.Errorf("%v (tag ref check: %v)", err, refErr))
		}
		// 通知在部署成功後才跑；tag 沒有對應 Release 只是少了 release notes，不該讓 build 變 FAILURE
		fmt.Fprintf(os.Stderr, "WARN: release not found for tag %s, notify without release notes: %v\n", inputTag, err)
		releaseURL = githubClient.GetReleaseURL(inputOwner, inputRepo, inputTag)
		releaseBody = noReleaseNotesText
	}

	if inputSlackChannel != "" && inputSlackWebhook != "" {

		// Set inputProject to inputRepo if inputProject is not set
		if inputProject == "" {
			inputProject = inputRepo
		}
	
		text := fmt.Sprintf(
			"%s deployment successful. \nProject: %s \nVersion: %s \nDomain: %s \n<%s|Click here> for details! \n\n%s",
			inputRepo,
			inputProject,
			inputTag,
			inputDomain,
			releaseURL,
			releaseBody,
		)

		if dryRun {
			fmt.Println(text)
		} else {
			sendSlackNotification(text)
		}
	} else if err == nil {
		// 沒有 Release 時 WARN 已寫到 stderr；stdout 只放真正的 release body，避免呼叫端把 fallback 文字當內容
		fmt.Println(releaseBody)
	}
}

func isNotFound(err error) bool {
	errResp, ok := err.(*gogithub.ErrorResponse)
	return ok && errResp.Response != nil && errResp.Response.StatusCode == http.StatusNotFound
}

func sendSlackNotification(text string) {
	msg := &slack.WebhookMessage{
		Channel: inputSlackChannel,
		Text:    text,
	}

	err := slack.PostWebhook(inputSlackWebhook, msg)

	if err != nil {
		fmt.Printf("Failed to send slack notification: %v\n", err)
	}
}
