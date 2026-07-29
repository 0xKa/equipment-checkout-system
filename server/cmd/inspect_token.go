package cmd

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const tokenInspectionWarning = `Inspection only: "Expired?" compares exp with this machine's clock. Decoding does not validate the signature, issuer, audience, or application access. The API performs those checks.`

type accessTokenClaims struct {
	Issuer            string                    `json:"iss"`
	Subject           string                    `json:"sub"`
	Audience          json.RawMessage           `json:"aud"`
	AuthorizedParty   string                    `json:"azp"`
	PreferredUsername string                    `json:"preferred_username"`
	EmailVerified     *bool                     `json:"email_verified"`
	ResourceAccess    map[string]resourceAccess `json:"resource_access"`
	ExpiresAt         *int64                    `json:"exp"`
}

type resourceAccess struct {
	Roles []string `json:"roles"`
}

type accessTokenMetadata struct {
	Issuer            string
	Subject           string
	Audience          string
	AuthorizedParty   string
	PreferredUsername string
	EmailVerified     string
	EquipmentAPIRoles string
	ExpiresAtUTC      string
	Expired           string
}

func newInspectTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect-token",
		Short: "Decode allowlisted metadata from a Keycloak access token",
		Args:  cobra.NoArgs,
		RunE:  runInspectToken,
	}
}

func runInspectToken(command *cobra.Command, _ []string) error {
	token, err := readAccessToken(command.InOrStdin(), command.ErrOrStderr())
	if err != nil {
		return err
	}

	metadata, err := decodeAccessTokenMetadata(token)
	if err != nil {
		return err
	}

	writeAccessTokenMetadata(command.OutOrStdout(), metadata)
	fmt.Fprintf(command.ErrOrStderr(), "\nWarning: %s\n", tokenInspectionWarning)
	return nil
}

func readAccessToken(input io.Reader, promptWriter io.Writer) (string, error) {
	if inputFile, ok := input.(*os.File); ok && term.IsTerminal(int(inputFile.Fd())) {
		fmt.Fprint(promptWriter, "Access token (input is hidden): ")

		tokenBytes, err := term.ReadPassword(int(inputFile.Fd()))
		fmt.Fprintln(promptWriter)
		if err != nil {
			return "", fmt.Errorf("read access token: %w", err)
		}
		defer clear(tokenBytes)

		return requireAccessToken(string(tokenBytes))
	}

	token, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read access token: %w", err)
	}

	return requireAccessToken(token)
}

func requireAccessToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("access token is required")
	}

	return token, nil
}

func decodeAccessTokenMetadata(token string) (accessTokenMetadata, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return accessTokenMetadata{}, errors.New("the value is not a compact JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return accessTokenMetadata{}, errors.New("the JWT payload is not valid base64url")
	}

	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return accessTokenMetadata{}, errors.New("the JWT payload is not valid JSON")
	}

	audience, err := decodeAudience(claims.Audience)
	if err != nil {
		return accessTokenMetadata{}, err
	}

	metadata := accessTokenMetadata{
		Issuer:            claims.Issuer,
		Subject:           claims.Subject,
		Audience:          strings.Join(audience, ", "),
		AuthorizedParty:   claims.AuthorizedParty,
		PreferredUsername: claims.PreferredUsername,
		EquipmentAPIRoles: strings.Join(claims.ResourceAccess["equipment-api"].Roles, ", "),
		Expired:           "unknown (missing exp)",
	}

	if claims.EmailVerified != nil {
		metadata.EmailVerified = fmt.Sprintf("%t", *claims.EmailVerified)
	}
	if claims.ExpiresAt != nil {
		expiresAt := time.Unix(*claims.ExpiresAt, 0).UTC()
		metadata.ExpiresAtUTC = expiresAt.Format(time.RFC3339Nano)
		metadata.Expired = fmt.Sprintf("%t", !time.Now().Before(expiresAt))
	}

	return metadata, nil
}

func decodeAudience(rawAudience json.RawMessage) ([]string, error) {
	if len(rawAudience) == 0 || string(rawAudience) == "null" {
		return nil, nil
	}

	var singleAudience string
	if err := json.Unmarshal(rawAudience, &singleAudience); err == nil {
		return []string{singleAudience}, nil
	}

	var multipleAudiences []string
	if err := json.Unmarshal(rawAudience, &multipleAudiences); err != nil {
		return nil, errors.New("the JWT audience claim must be a string or an array of strings")
	}

	return multipleAudiences, nil
}

func writeAccessTokenMetadata(output io.Writer, metadata accessTokenMetadata) {
	fmt.Fprintln(output, "Access Token Metadata")
	fmt.Fprintln(output, "=====================")

	fmt.Fprintln(output, "\nIdentity")
	fmt.Fprintln(output, "--------")
	writeMetadataField(output, "Issuer", metadata.Issuer)
	writeMetadataField(output, "Subject", metadata.Subject)
	writeMetadataField(output, "Preferred username", metadata.PreferredUsername)
	writeMetadataField(output, "Email verified", metadata.EmailVerified)

	fmt.Fprintln(output, "\nAuthorization")
	fmt.Fprintln(output, "-------------")
	writeMetadataField(output, "Audience", metadata.Audience)
	writeMetadataField(output, "Authorized party", metadata.AuthorizedParty)
	writeMetadataField(output, "Equipment API roles", metadata.EquipmentAPIRoles)

	fmt.Fprintln(output, "\nExpiration")
	fmt.Fprintln(output, "----------")
	writeMetadataField(output, "Expires at (UTC)", metadata.ExpiresAtUTC)
	writeMetadataField(output, "Expired?", metadata.Expired)
}

func writeMetadataField(output io.Writer, label string, value string) {
	if value == "" {
		value = "-"
	}

	fmt.Fprintf(output, "  %-20s : %s\n", label, value)
}
