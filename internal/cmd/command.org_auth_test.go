package cmd_test

import (
	"testing"
)

func orgInfoFixture(roleArn, externalID any) map[string]any {
	return map[string]any{
		"org_id":                       "abcd1234",
		"aws_customer_access_role_arn": roleArn,
		"aws_external_id":              externalID,
	}
}

func Test_Org_AwsAssumeRole_Text(t *testing.T) {
	h := NewCommandHarness(t)
	h.MockManagementAPI().SetRoute("GET", "/v1/organizations/me", 200,
		orgInfoFixture(
			"arn:aws:iam::337139236424:role/baseten-customer-access",
			"baseten-2fdd8a01c4c34e6bb92a2b96fca29b70",
		))

	h.Require.NoError(h.Execute("org", "aws-assume-role"))
	out := h.Stdout.String()
	h.Require.Contains(out, "Baseten Role ARN: arn:aws:iam::337139236424:role/baseten-customer-access")
	h.Require.Contains(out, "AWS External ID:  baseten-2fdd8a01c4c34e6bb92a2b96fca29b70")
}

func Test_Org_AwsAssumeRole_JSON(t *testing.T) {
	h := NewCommandHarness(t)
	h.MockManagementAPI().SetRoute("GET", "/v1/organizations/me", 200,
		orgInfoFixture(
			"arn:aws:iam::337139236424:role/baseten-customer-access",
			"baseten-2fdd8a01c4c34e6bb92a2b96fca29b70",
		))

	h.Require.NoError(h.Execute("org", "aws-assume-role", "--output", "json"))
	h.Require.Contains(h.Stdout.String(), `"external_id": "baseten-2fdd8a01c4c34e6bb92a2b96fca29b70"`)
}

func Test_Org_AwsAssumeRole_NotEnabled(t *testing.T) {
	h := NewCommandHarness(t)
	h.MockManagementAPI().SetRoute("GET", "/v1/organizations/me", 200,
		orgInfoFixture(nil, nil))

	err := h.Execute("org", "aws-assume-role")
	h.Require.ErrorContains(err, "not enabled for this organization")
}

func Test_Org_Oidc_Text(t *testing.T) {
	h := NewCommandHarness(t)
	h.MockManagementAPI().SetRoute("GET", "/v1/organizations/me", 200,
		orgInfoFixture(nil, nil))
	h.MockManagementAPI().SetRoute("GET", "/v1/teams", 200,
		map[string]any{"teams": []any{
			map[string]any{
				"id": "t1", "name": "my-team", "default": true,
				"created_at": "2026-01-01T00:00:00Z",
			},
		}})

	h.Require.NoError(h.Execute("org", "oidc"))
	out := h.Stdout.String()
	h.Require.Contains(out, "Org ID:               abcd1234")
	h.Require.Contains(out, "t1 (my-team)")
	h.Require.Contains(out, "Issuer:               https://oidc.baseten.co")
	h.Require.Contains(out, "Subject Claim Format: v=1:org=<org_id>")
}

func Test_Org_Oidc_JSON(t *testing.T) {
	h := NewCommandHarness(t)
	h.MockManagementAPI().SetRoute("GET", "/v1/organizations/me", 200,
		orgInfoFixture(nil, nil))
	h.MockManagementAPI().SetRoute("GET", "/v1/teams", 200,
		map[string]any{"teams": []any{}})

	h.Require.NoError(h.Execute("org", "oidc", "--output", "json"))
	h.Require.Contains(h.Stdout.String(), `"org_id": "abcd1234"`)
	h.Require.Contains(h.Stdout.String(), `"issuer": "https://oidc.baseten.co"`)
}
