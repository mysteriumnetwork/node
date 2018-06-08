/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package terms_and_conditions

const optionAgreeValue = "04-06-2018"

// UserAgreed returns whether user provided agreement string means user agreed with terms & conditions
func UserAgreed(agreementString string) bool {
	return agreementString == optionAgreeValue
}

// Text returns terms & conditions with explanation on how to agree with them
var Text = termsAndConditions + "\n\n" + explanationString
