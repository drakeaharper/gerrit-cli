package cmd

import (
	"reflect"
	"testing"

	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
)

func jenkinsMsg(body string) gerrit.ChangeMessageInfo {
	return gerrit.ChangeMessageInfo{
		Author:  gerrit.Account{Name: "Service Cloud Jenkins"},
		Message: body,
	}
}

func TestFindMostRecentFailure(t *testing.T) {
	multiSection := "Patch Set 1: Verified-1\n\nBuild Failed /o\\\n" +
		"[Build summary report](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build-summary-report/)\n\n" +
		"Test failures:\n\n" +
		"- [Vitest 07 - ForceFailure > a failure is not forced](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/test?L=1)\n\n" +
		"Build failures:\n\n" +
		"- [RspecQ Tests - Reporter](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build?L=2)"

	tests := []struct {
		name     string
		messages []gerrit.ChangeMessageInfo
		want     failureResult
	}{
		{
			name:     "multiple failure sections",
			messages: []gerrit.ChangeMessageInfo{jenkinsMsg(multiSection)},
			want: failureResult{
				SummaryLink: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build-summary-report/",
				Sections: []failureSection{
					{
						Title: "Test failures",
						Failures: []buildFailure{
							{Name: "Vitest 07 - ForceFailure > a failure is not forced", Link: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169/test?L=1"},
						},
					},
					{
						Title: "Build failures",
						Failures: []buildFailure{
							{Name: "RspecQ Tests - Reporter", Link: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build?L=2"},
						},
					},
				},
			},
		},
		{
			name: "single build failures section",
			messages: []gerrit.ChangeMessageInfo{
				jenkinsMsg("Patch Set 1: Verified-1\n\nBuild Failed /o\\\n" +
					"[Build summary report](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build-summary-report/)\n\n" +
					"Build failures:\n- [Build Docker Image](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/artifact/log.html?abc)"),
			},
			want: failureResult{
				SummaryLink: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build-summary-report/",
				Sections: []failureSection{
					{
						Title: "Build failures",
						Failures: []buildFailure{
							{Name: "Build Docker Image", Link: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169/artifact/log.html?abc"},
						},
					},
				},
			},
		},
		{
			name: "plaintext link with double slash (legacy format), no sections",
			messages: []gerrit.ChangeMessageInfo{
				jenkinsMsg("Patch Set 1: Verified-1\n\nBuild Failed\n\nhttps://jenkins.inst-ci.net/job/Canvas/job/main/191169//build-summary-report/ : FAILURE"),
			},
			want: failureResult{
				SummaryLink: "https://jenkins.inst-ci.net/job/Canvas/job/main/191169//build-summary-report/",
			},
		},
		{
			name: "ignores messages without Verified-1",
			messages: []gerrit.ChangeMessageInfo{
				jenkinsMsg("Patch Set 1:\n\nBuild Started https://jenkins.inst-ci.net/job/Canvas/job/main/191169/"),
			},
			want: failureResult{},
		},
		{
			name: "ignores non-jenkins authors",
			messages: []gerrit.ChangeMessageInfo{
				{
					Author:  gerrit.Account{Name: "Drake Harper"},
					Message: "Patch Set 1: Verified-1\n[Build summary report](https://jenkins.inst-ci.net/job/Canvas/job/main/191169/build-summary-report/)",
				},
			},
			want: failureResult{},
		},
		{
			name: "returns most recent failure when multiple present",
			messages: []gerrit.ChangeMessageInfo{
				jenkinsMsg("Patch Set 1: Verified-1\n[Build summary report](https://jenkins.inst-ci.net/job/Canvas/job/main/100/build-summary-report/)"),
				jenkinsMsg("Patch Set 2: Verified-1\n[Build summary report](https://jenkins.inst-ci.net/job/Canvas/job/main/200/build-summary-report/)"),
			},
			want: failureResult{
				SummaryLink: "https://jenkins.inst-ci.net/job/Canvas/job/main/200/build-summary-report/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMostRecentFailure(tt.messages)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findMostRecentFailure() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
