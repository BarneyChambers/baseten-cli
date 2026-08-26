package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/basetenlabs/baseten-cli/cmd"
	"github.com/basetenlabs/baseten-go/client/managementapi"
)

func init() {
	Register("org oidc", commandOrgOidc)
	Register("org aws-assume-role", commandOrgAwsAssumeRole)
}

const (
	oidcIssuer             = "https://oidc.baseten.co"
	oidcAudience           = "oidc.baseten.co"
	oidcSubjectClaimFormat = "v=1:org=<org_id>:team=<team_id>:model=<model_id>:" +
		"deployment=<deployment_id>:environment=<environment>:type=<workload_type>"
)

// organizationInfo mirrors GET /v1/organization. Fetched raw and kept local to
// this package until the endpoint lands in baseten-go's generated client.
type organizationInfo struct {
	OrgID string `json:"org_id"`
	// Both fields are null while AWS AssumeRole is not enabled for the org.
	AwsCustomerAccessRoleArn *string `json:"aws_customer_access_role_arn"`
	AwsExternalID            *string `json:"aws_external_id"`
}

func getOrganizationInfo(ctx *CommandContext, api *managementapi.Client) (*organizationInfo, error) {
	url := strings.TrimRight(api.BaseURL, "/") + "/v1/organization"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for key, vals := range api.Headers {
		for _, val := range vals {
			req.Header.Add(key, val)
		}
	}
	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching organization info: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading organization info: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &managementapi.ResponseError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var info organizationInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding organization info: %w", err)
	}
	return &info, nil
}

func commandOrgOidc(ctx *CommandContext, flags *cmd.OrgOidcFlags) error {
	cl, err := ctx.NewManagementClient()
	if err != nil {
		return err
	}
	info, err := getOrganizationInfo(ctx, cl.API())
	if err != nil {
		return err
	}
	teams, err := cl.API().GetTeams(ctx, managementapi.GetV1TeamsParams{})
	if err != nil {
		return fmt.Errorf("listing teams: %w", err)
	}

	out := cmd.OrgOidcInfo{
		OrgID:              info.OrgID,
		Issuer:             oidcIssuer,
		Audience:           oidcAudience,
		WorkloadTypes:      []string{"model_container", "model_build"},
		SubjectClaimFormat: oidcSubjectClaimFormat,
	}
	for _, t := range teams.Teams {
		out.Teams = append(out.Teams, cmd.OrgOidcTeam{ID: t.Id, Name: t.Name})
	}

	if ctx.JSON {
		ctx.OutputJSON(out)
		return nil
	}
	ctx.Outputf("Org ID:               %s\n", out.OrgID)
	for i, t := range out.Teams {
		label := "Teams:               "
		if i > 0 {
			label = "                     "
		}
		ctx.Outputf("%s %s (%s)\n", label, t.ID, t.Name)
	}
	ctx.Outputf("Issuer:               %s\n", out.Issuer)
	ctx.Outputf("Audience:             %s\n", out.Audience)
	ctx.Outputf("Workload Types:       %s\n", strings.Join(out.WorkloadTypes, ", "))
	ctx.Outputf("Subject Claim Format: %s\n", out.SubjectClaimFormat)
	return nil
}

func commandOrgAwsAssumeRole(ctx *CommandContext, flags *cmd.OrgAwsAssumeRoleFlags) error {
	cl, err := ctx.NewManagementClient()
	if err != nil {
		return err
	}
	info, err := getOrganizationInfo(ctx, cl.API())
	if err != nil {
		return err
	}
	// The server nulls both fields while the method is not enabled for the org.
	if info.AwsCustomerAccessRoleArn == nil || info.AwsExternalID == nil {
		return fmt.Errorf(
			"AWS AssumeRole is not enabled for this organization; contact Baseten support to enable it",
		)
	}

	out := cmd.OrgAwsAssumeRoleInfo{
		RoleArn:    *info.AwsCustomerAccessRoleArn,
		ExternalID: *info.AwsExternalID,
	}
	if ctx.JSON {
		ctx.OutputJSON(out)
		return nil
	}
	ctx.Outputf("Baseten Role ARN: %s\n", out.RoleArn)
	ctx.Outputf("AWS External ID:  %s\n", out.ExternalID)
	return nil
}
