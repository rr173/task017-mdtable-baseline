package mdtable

import (
	"strings"
	"testing"
)

func TestFormatBasic(t *testing.T) {
	header := []string{"Name", "Age"}
	rows := [][]string{{"Alice", "30"}, {"Bob", "25"}}
	aligns := []Alignment{AlignLeft, AlignRight}
	got, widths, err := Format(header, rows, aligns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "| Name  | Age |\n| :---- | --: |\n| Alice |  30 |\n| Bob   |  25 |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
	if len(widths) != 2 || widths[0] != 5 || widths[1] != 3 {
		t.Errorf("widths = %v, want [5 3]", widths)
	}
}

func TestFormatDefaultAligns(t *testing.T) {
	// aligns 为 nil 时全部默认对齐；列宽下限 3，分隔为 ---。
	got, _, err := Format([]string{"a", "b"}, [][]string{{"1", "2"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "| a   | b   |\n| --- | --- |\n| 1   | 2   |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatCJKWidth(t *testing.T) {
	header := []string{"项目", "数量"}
	rows := [][]string{{"苹果", "5"}, {"香蕉", "12"}}
	got, widths, err := Format(header, rows, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 中文每字宽度 2，列宽为 4；数字列宽度 max(4,1,2)=4。
	if widths[0] != 4 || widths[1] != 4 {
		t.Errorf("widths = %v, want [4 4]", widths)
	}
	want := "| 项目 | 数量 |\n| ---- | ---- |\n| 苹果 | 5    |\n| 香蕉 | 12   |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"中", 2},
		{"中文", 4},
		{"a中b", 4}, // 1+2+1
		{`a\b`, 3}, // 反斜杠算 1：a+\+b
		{"\t\n", 0},
		{"中a", 3}, // 2+1
	}
	for _, c := range cases {
		if got := DisplayWidth(c.in); got != c.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestEscapePipe(t *testing.T) {
	// 单元格内的竖线必须转义为 \|，不得被误判为列分隔符。
	got, _, err := Format([]string{"a", "b"}, [][]string{{"x|y", "z"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, `x\|y`) {
		t.Errorf("pipe not escaped: %q", got)
	}
	// 按未转义的竖线切分，每行应恰为 2 列（转义后的竖线不构成列边界）。
	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		cols := splitColumns(line)
		if len(cols) != 2 {
			t.Errorf("line %q split into %d columns, want 2", line, len(cols))
		}
	}
}

func TestEscapeBackslash(t *testing.T) {
	// 反斜杠必须转义为 \\，且先于竖线处理，避免二次转义。
	got, _, err := Format([]string{"a"}, [][]string{{`a\b|c`}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 原始 a\b|c → 转义后 a\\b\|c
	if !strings.Contains(got, `a\\b\|c`) {
		t.Errorf("backslash/pipe not escaped correctly: %q", got)
	}
}

func TestNewlineRejected(t *testing.T) {
	if _, _, err := Format([]string{"a\nb"}, nil, nil); err == nil {
		t.Error("header with newline should be rejected")
	}
	if _, _, err := Format([]string{"a"}, [][]string{{"x\ny"}}, nil); err == nil {
		t.Error("cell with newline should be rejected")
	}
	if err := Validate([]string{"a"}, [][]string{{"x\ry"}}, nil); err == nil {
		t.Error("cell with CR should be rejected")
	}
}

func TestRowTooLongRejected(t *testing.T) {
	err := Validate([]string{"a", "b"}, [][]string{{"1", "2", "3"}}, nil)
	if err == nil {
		t.Error("row longer than header should be rejected")
	}
}

func TestRowShortPadded(t *testing.T) {
	// 少于表头列数的数据行应补齐而非报错。
	got, _, err := Format([]string{"a", "b", "c"}, [][]string{{"1"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "| a   | b   | c   |\n| --- | --- | --- |\n| 1   |     |     |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestEmptyHeaderRejected(t *testing.T) {
	if _, _, err := Format(nil, nil, nil); err == nil {
		t.Error("empty header should be rejected")
	}
	if _, _, err := Format([]string{}, nil, nil); err == nil {
		t.Error("empty header should be rejected")
	}
}

func TestAlignsLengthMismatch(t *testing.T) {
	if err := Validate([]string{"a", "b"}, nil, []Alignment{AlignLeft}); err == nil {
		t.Error("aligns length mismatch should be rejected")
	}
}

func TestHeaderOnly(t *testing.T) {
	// 仅有表头无数据行仍应输出合法表格。
	got, _, err := Format([]string{"h1", "h2"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "| h1  | h2  |\n| --- | --- |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestSingleColumn(t *testing.T) {
	got, _, err := Format([]string{"h"}, [][]string{{"x"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "| h   |\n| --- |\n| x   |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestCenterAlign(t *testing.T) {
	got, _, err := Format([]string{"h"}, [][]string{{"x"}}, []Alignment{AlignCenter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 单列宽度下限 3，居中对齐分隔为 ":-:"。
	want := "|  h  |\n| :-: |\n|  x  |\n"
	if got != want {
		t.Errorf("table mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestSeparatorWidthMatchesColumn(t *testing.T) {
	// 较宽列的分隔单元格总宽度必须等于该列宽度。
	got, widths, err := Format([]string{"header"}, [][]string{{"x"}}, []Alignment{AlignLeft})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	sep := lines[1]
	// 提取分隔单元格内容（去掉首尾 "| " 与 " |"）。
	inner := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(sep, " |"), "| "))
	if displayWidth(inner) != widths[0] {
		t.Errorf("separator width %d != column width %d for %q", displayWidth(inner), widths[0], inner)
	}
	if !strings.HasPrefix(inner, ":") {
		t.Errorf("left-align separator should start with ':': %q", inner)
	}
}

func TestParseAlignment(t *testing.T) {
	cases := []struct {
		in   string
		want Alignment
	}{
		{"", AlignDefault},
		{"default", AlignDefault},
		{"LEFT", AlignLeft},
		{" center ", AlignCenter},
		{"right", AlignRight},
	}
	for _, c := range cases {
		got, err := ParseAlignment(c.in)
		if err != nil {
			t.Errorf("ParseAlignment(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseAlignment(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	if _, err := ParseAlignment("diagonal"); err == nil {
		t.Error("invalid alignment should error")
	}
}

// splitColumns 按未转义的竖线切分表格行，返回各单元格内容（含前后空白）。
// 转义后的 \| 视为单元格内的字面竖线，不构成列边界。仅用于测试断言。
func splitColumns(line string) []string {
	var cols []string
	var cur strings.Builder
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			cur.WriteRune('|')
			i++
			continue
		}
		if runes[i] == '|' {
			cols = append(cols, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(runes[i])
	}
	cols = append(cols, cur.String())
	// 行首行尾的竖线外侧为空串，去掉。
	for len(cols) >= 1 && cols[0] == "" {
		cols = cols[1:]
	}
	for len(cols) >= 1 && cols[len(cols)-1] == "" {
		cols = cols[:len(cols)-1]
	}
	return cols
}
