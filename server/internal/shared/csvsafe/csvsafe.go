// Package csvsafe keeps exported CSV cells from running as spreadsheet
// formulas (CSV injection).
package csvsafe

import "strings"

// Cell stops a spreadsheet from running a cell as a formula: text that
// starts with =, +, -, @ or a tab or carriage return gets a leading quote.
func Cell(value string) string {
	if value != "" && strings.ContainsRune("=+-@\t\r", rune(value[0])) {
		return "'" + value
	}
	return value
}
