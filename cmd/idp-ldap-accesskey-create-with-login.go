// Copyright (c) 2015-2023 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/pkg/v2/console"
	"golang.org/x/term"
)

var idpLdapAccesskeyCreateWithLoginCmd = cli.Command{
	Name:         "create-with-login",
	Usage:        "log in using LDAP credentials to generate access key pair",
	Action:       mainIDPLdapAccesskeyCreateWithLogin,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapAccesskeyCreateFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] URL

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new access key pair for https://minio.example.com by logging in with LDAP credentials
     {{.Prompt}} {{.HelpName}} https://minio.example.com
  2. Create a new access key pair for http://localhost:9000 via login with custom access key and secret key 
     {{.Prompt}} {{.HelpName}} http://localhost:9000 --access-key myaccesskey --secret-key mysecretkey
	`,
}

func mainIDPLdapAccesskeyCreateWithLogin(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	url := args.Get(0)

	opts := accessKeyCreateOpts(ctx, "")

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if !isTerminal {
		e := fmt.Errorf("login flag cannot be used with non-interactive terminal")
		fatalIf(probe.NewError(e), "Invalid flags.")
	}

	client := loginLDAPAccesskey(url)

	res, e := client.AddServiceAccountLDAP(globalContext, opts)
	fatalIf(probe.NewError(e), "Unable to add service account.")

	m := ldapAccesskeyMessage{
		op:          "create",
		Status:      "success",
		AccessKey:   res.AccessKey,
		SecretKey:   res.SecretKey,
		Expiration:  &res.Expiration,
		Name:        opts.Name,
		Description: opts.Description,
	}
	printMsg(m)

	return nil
}

func loginLDAPAccesskey(URL string) *madmin.AdminClient {
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s", console.Colorize(cred, "Enter LDAP Username: "))
	value, _, e := reader.ReadLine()
	fatalIf(probe.NewError(e), "Unable to read username")
	username := string(value)

	fmt.Printf("%s", console.Colorize(cred, "Enter Password: "))
	bytePassword, e := term.ReadPassword(int(os.Stdin.Fd()))
	fatalIf(probe.NewError(e), "Unable to read password")
	fmt.Printf("\n")
	password := string(bytePassword)

	ldapID, e := credentials.NewLDAPIdentity(URL, username, password)
	fatalIf(probe.NewError(e), "Unable to initialize LDAP identity.")

	u, e := url.Parse(URL)
	fatalIf(probe.NewError(e), "Unable to parse server URL.")

	client, e := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds:  ldapID,
		Secure: u.Scheme == "https",
	})
	fatalIf(probe.NewError(e), "Unable to initialize admin connection.")

	return client
}
