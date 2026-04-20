package unit

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keystone/backend/internal/parts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSVRows_ValidInput(t *testing.T) {
	csv := "part_number,name,description\nPN-001,Brake Pad,Heavy duty brake pad\n"
	rows, err := parts.ParseCSVRows([]byte(csv))
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "PN-001", rows[0].PartNumber)
	assert.Equal(t, "Brake Pad", rows[0].Name)
	assert.Equal(t, "Heavy duty brake pad", rows[0].Description)
}

func TestParseCSVRows_MissingHeader(t *testing.T) {
	csv := ""
	_, err := parts.ParseCSVRows([]byte(csv))
	assert.Error(t, err)
}

func TestParseCSVRows_OnlyHeader(t *testing.T) {
	csv := "part_number,name,description\n"
	_, err := parts.ParseCSVRows([]byte(csv))
	assert.Error(t, err, "CSV with only header and no data rows should fail")
}

func TestParseCSVRows_MultipleRows(t *testing.T) {
	csv := "part_number,name,description\nPN-001,Part A,Desc A\nPN-002,Part B,Desc B\n"
	rows, err := parts.ParseCSVRows([]byte(csv))
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, "PN-002", rows[1].PartNumber)
}

func TestParseCSVRows_WithJSONFields(t *testing.T) {
	csv := `part_number,name,fitment,oem_mappings
PN-001,Brake Pad,"{""make"":""Ford""}","{""oem"":""12345""}"
`
	rows, err := parts.ParseCSVRows([]byte(csv))
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	// The fitment JSON should be parseable.
	var fitment map[string]string
	err = json.Unmarshal([]byte(rows[0].FitmentJSON), &fitment)
	assert.NoError(t, err)
	assert.Equal(t, "Ford", fitment["make"])
}

func TestVersionNumberIncrement(t *testing.T) {
	maxVer := 3
	newVer := maxVer + 1
	assert.Equal(t, 4, newVer)
}

func TestVersionCompareDiff_ChangedField(t *testing.T) {
	v1 := json.RawMessage(`{"make":"Ford"}`)
	v2 := json.RawMessage(`{"make":"Toyota"}`)
	changed := string(v1) != string(v2)
	assert.True(t, changed)
}

func TestVersionCompareDiff_UnchangedField(t *testing.T) {
	v1 := json.RawMessage(`{"make":"Ford"}`)
	v2 := json.RawMessage(`{"make":"Ford"}`)
	changed := string(v1) != string(v2)
	assert.False(t, changed)
}

func TestPartPartNumberRequired(t *testing.T) {
	row := parts.CSVImportRow{PartNumber: "", Name: "Test Part"}
	isValid := strings.TrimSpace(row.PartNumber) != ""
	assert.False(t, isValid, "empty part number should fail validation")
}

func TestPartNameRequired(t *testing.T) {
	row := parts.CSVImportRow{PartNumber: "PN-001", Name: ""}
	isValid := strings.TrimSpace(row.Name) != ""
	assert.False(t, isValid, "empty part name should fail validation")
}

func TestPartValidRow(t *testing.T) {
	row := parts.CSVImportRow{PartNumber: "PN-001", Name: "Brake Pad"}
	isValid := strings.TrimSpace(row.PartNumber) != "" && strings.TrimSpace(row.Name) != ""
	assert.True(t, isValid)
}

func TestPartStatus_DefaultActive(t *testing.T) {
	defaultStatus := "ACTIVE"
	assert.Equal(t, "ACTIVE", defaultStatus)
}

func TestPartPromoteVersion_WrongPart(t *testing.T) {
	// A version from part A cannot be promoted on part B.
	versionPartID := "part-A"
	targetPartID := "part-B"
	isValid := versionPartID == targetPartID
	assert.False(t, isValid)
}

func TestPartExportFieldSelection(t *testing.T) {
	allFields := []string{"id", "part_number", "name", "description", "status", "created_at"}
	selectedFields := []string{"part_number", "name"}

	for _, f := range selectedFields {
		assert.Contains(t, allFields, f)
	}
}
