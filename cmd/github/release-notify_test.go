package github

import (
	"deploy-helper/components/aws/github"
	"deploy-helper/components/aws/mocks"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
)

const (
	testOwner      = "kkday-it"
	testRepo       = "kkday-b2c-api"
	testTag        = "v2.55.0-rc.2"
	testReleaseURL = "https://github.com/kkday-it/kkday-b2c-api/releases/tag/v2.55.0-rc.2"
)

func newErrorResponse(statusCode int) error {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/kkday-it/kkday-b2c-api/releases/tags/v2.55.0-rc.2", nil)
	return &gogithub.ErrorResponse{Response: &http.Response{StatusCode: statusCode, Request: req}}
}

func setupReleaseNotify(t *testing.T, release *gogithub.RepositoryRelease, err error, tagRefErr error, withSlack bool) *mocks.IfaceGithub {
	client := &mocks.IfaceGithub{}
	client.On("GetReleaseByTag", testOwner, testRepo, testTag).Return(release, err)
	client.On("GetTagRef", testOwner, testRepo, testTag).Return(&gogithub.Reference{}, tagRefErr)
	client.On("GetReleaseURL", testOwner, testRepo, testTag).Return(testReleaseURL)

	origFactory := newGithubClient
	origOwner, origRepo, origTag := inputOwner, inputRepo, inputTag
	origProject, origDomain := inputProject, inputDomain
	origSlackWebhook, origSlackChannel, origDryRun := inputSlackWebhook, inputSlackChannel, dryRun
	t.Cleanup(func() {
		newGithubClient = origFactory
		inputOwner, inputRepo, inputTag = origOwner, origRepo, origTag
		inputProject, inputDomain = origProject, origDomain
		inputSlackWebhook, inputSlackChannel, dryRun = origSlackWebhook, origSlackChannel, origDryRun
	})

	newGithubClient = func(token string) github.IfaceGithub { return client }
	inputOwner, inputRepo, inputTag = testOwner, testRepo, testTag
	inputProject, inputDomain = "", "api.kkday.com"
	inputSlackWebhook, inputSlackChannel = "", ""
	if withSlack {
		inputSlackWebhook, inputSlackChannel = "https://hooks.slack.invalid", "#deploy"
	}
	dryRun = true

	return client
}

// captureOutput 回傳執行 fn 期間寫到 stdout、stderr 的內容
func captureOutput(t *testing.T, fn func()) (string, string) {
	origStdout, origStderr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = origStdout, origStderr }()

	fn()

	outW.Close()
	errW.Close()
	stdout, _ := ioutil.ReadAll(outR)
	stderr, _ := ioutil.ReadAll(errR)
	return string(stdout), string(stderr)
}

func TestReleaseNotifyWithRelease(t *testing.T) {
	release := &gogithub.RepositoryRelease{
		HTMLURL: gogithub.String("https://github.com/kkday-it/kkday-b2c-api/releases/tag/v2.55.0-rc.2"),
		Body:    gogithub.String("- fix something"),
	}
	client := setupReleaseNotify(t, release, nil, nil, true)

	stdout, stderr := captureOutput(t, func() { runReleaseNotifyCmd(nil, nil) })

	expected := "kkday-b2c-api deployment successful. \nProject: kkday-b2c-api \nVersion: v2.55.0-rc.2 \nDomain: api.kkday.com \n" +
		"<https://github.com/kkday-it/kkday-b2c-api/releases/tag/v2.55.0-rc.2|Click here> for details! \n\n- fix something\n"
	assert.Equal(t, expected, stdout)
	assert.Equal(t, "", stderr)
	// 有 Release 的正常部署不該多打 tag ref API
	client.AssertNotCalled(t, "GetTagRef", testOwner, testRepo, testTag)
}

func TestReleaseNotifyReleaseNotFound(t *testing.T) {
	setupReleaseNotify(t, nil, newErrorResponse(http.StatusNotFound), nil, true)

	stdout, stderr := captureOutput(t, func() { runReleaseNotifyCmd(nil, nil) })

	expected := "kkday-b2c-api deployment successful. \nProject: kkday-b2c-api \nVersion: v2.55.0-rc.2 \nDomain: api.kkday.com \n" +
		"<" + testReleaseURL + "|Click here> for details! \n\nNo release found for this tag. A newer release candidate may have replaced it.\n"
	assert.Equal(t, expected, stdout)
	assert.True(t, strings.HasPrefix(stderr, "WARN: release not found for tag v2.55.0-rc.2"), stderr)
}

func TestReleaseNotifyReleaseNotFoundWithoutSlack(t *testing.T) {
	setupReleaseNotify(t, nil, newErrorResponse(http.StatusNotFound), nil, false)

	stdout, stderr := captureOutput(t, func() { runReleaseNotifyCmd(nil, nil) })

	assert.Equal(t, "", stdout)
	assert.True(t, strings.HasPrefix(stderr, "WARN: release not found for tag v2.55.0-rc.2"), stderr)
}

func TestReleaseNotifyOtherErrorPanics(t *testing.T) {
	setupReleaseNotify(t, nil, newErrorResponse(http.StatusUnauthorized), nil, true)

	assert.Panics(t, func() { runReleaseNotifyCmd(nil, nil) })
}

func TestReleaseNotifyRepoNotReadablePanics(t *testing.T) {
	setupReleaseNotify(t, nil, newErrorResponse(http.StatusNotFound), newErrorResponse(http.StatusNotFound), true)

	assert.Panics(t, func() { runReleaseNotifyCmd(nil, nil) })
}
