/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package config

import (
	"github.com/urfave/cli/v2"
)

// FlagQUICDomain defines domain to be used by the QUIC connections.
var FlagQUICDomain = cli.StringFlag{
	Name:  "quic.domain",
	Usage: "Domain to be used by the QUIC service",
	Value: "",
}

// FlagQUICKey defines key to be used by the QUIC service.
var FlagQUICKey = cli.StringFlag{
	Name:  "quic.key",
	Usage: "Key to be used by the QUIC service",
	Value: "",
}

// FlagQUICCert defines cert to be used by the QUIC service.
var FlagQUICCert = cli.StringFlag{
	Name:  "quic.cert",
	Usage: "Cert to be used by the QUIC service",
	Value: "",
}

// FlagQUICLogin defines login to be used by the QUIC service.
var FlagQUICLogin = cli.StringFlag{
	Name:  "quic.login",
	Usage: "Login to be used by the QUIC service",
	Value: "mystuser",
}

// FlagQUICPassword defines password to be used by the QUIC service.
var FlagQUICPassword = cli.StringFlag{
	Name:  "quic.password",
	Usage: "Password to be used by the QUIC service",
	Value: "mystpass",
}

// RegisterFlagsServiceQuic function register QUIC flags to flag list.
func RegisterFlagsServiceQuic(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagQUICDomain,
		&FlagQUICKey,
		&FlagQUICCert,
		&FlagQUICLogin,
		&FlagQUICPassword,
	)
}

// ParseFlagsServiceQuic parses CLI flags and registers value to configuration.
func ParseFlagsServiceQuic(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagQUICDomain)
	Current.ParseStringFlag(ctx, FlagQUICKey)
	Current.ParseStringFlag(ctx, FlagQUICCert)
	Current.ParseStringFlag(ctx, FlagQUICLogin)
	Current.ParseStringFlag(ctx, FlagQUICPassword)
}
