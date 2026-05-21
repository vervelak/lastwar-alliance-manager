package main

// mg_ocr.go — OCR pipeline for Marshal Guard screenshots (v2).
// Ported from mg_segment/main.go.  All types/functions are prefixed with "mg"
// to avoid collisions with the existing identifiers in main.go.

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	gosseract "github.com/otiai10/gosseract/v2"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type mgRect struct{ x0, y0, x1, y1 int }

type mgMemberOCR struct {
	Rank      int     // reconstructed rank (2–100)
	Name      string  // e.g. "[RSRP]Gargoland"
	NameOK    bool    // name OCR was validated
	DamageStr string  // e.g. "27.35G"
	DamageInt int64   // parsed value in bytes/units
	DamageOK  bool    // damage OCR was validated
	RankFixed bool    // rank was inferred (CORR/FIXD)
	FileIdx   int     // index of the source uploaded file (set by caller)
	CropY0    float64 // row top relative to full image (0.0–1.0)
	CropY1    float64 // row bottom relative to full image (0.0–1.0)
}

// MGImgResult holds all data extracted from one Marshal Guard screenshot.
type MGImgResult struct {
	EventDate       string // "2026-05-06"
	TopPlayerName   string // "[RSRP]Gargoland" — empty if not found
	TopPlayerDmgStr string // "27.35G" — empty if not found
	TopPlayerDmgInt int64
	Members         []mgMemberOCR
}

// ─── Package-level regexes (MG-specific) ─────────────────────────────────────

var (
	mgRankRe     = regexp.MustCompile(`^([2-9]|[1-9][0-9]|100)$`)
	mgNameRe     = regexp.MustCompile(`^\[[A-Za-z0-9]{1,4}\]\s*([A-Za-z0-9 ]+)$`)
	mgDamageRe   = regexp.MustCompile(`^Total Damage:\s\d+(?:\.\d{1,2})?[GM]$`)
	mgDatetimeRe = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}\s+\d{2}:\d{2}:\d{2}$`)
)

// ─── Image processing helpers ─────────────────────────────────────────────────

// mgFindDialogBounds detects the bounding box of the main dialog popup.
func mgFindDialogBounds(img image.Image, width, height int) (x0, y0, x1, y1 int) {
	sampleY0 := height / 3
	sampleY1 := height * 2 / 3
	colMean := make([]int, width)
	for x := 0; x < width; x++ {
		sum, count := 0, 0
		for y := sampleY0; y < sampleY1; y += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMean[x] = sum / count
		}
	}
	x0 = 0
	for x := 0; x < width; x++ {
		if colMean[x] > 170 {
			x0 = x
			break
		}
	}
	x1 = width
	for x := width - 1; x >= 0; x-- {
		if colMean[x] > 170 {
			x1 = x + 1
			break
		}
	}
	lightThreshold := 200
	y0 = 0
	for y := 0; y < height; y++ {
		if mgRowMeanInRange(img, y, x0, x1) > lightThreshold {
			y0 = y
			break
		}
	}
	y1 = height
	for y := height - 1; y >= 0; y-- {
		if mgRowMeanInRange(img, y, x0, x1) > lightThreshold {
			y1 = y + 1
			break
		}
	}
	return x0, y0, x1, y1
}

func mgRowMeanInRange(img image.Image, y, x0, x1 int) int {
	sum, count := 0, 0
	for x := x0; x < x1; x += 5 {
		r, g, b, _ := img.At(x, y).RGBA()
		sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

// mgFindVerticalSegments scans a rectangle left to right and returns content
// (non-background) column-strip bounding boxes.
func mgFindVerticalSegments(img image.Image, rect mgRect) []mgRect {
	rectW := rect.x1 - rect.x0
	if rectW < 16 {
		return nil
	}
	colMeans := make([]int, rectW)
	for i := 0; i < rectW; i++ {
		x := rect.x0 + i
		sum, count := 0, 0
		for y := rect.y0; y < rect.y1; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMeans[i] = sum / count
		}
	}
	refSum := 0
	for i := 5; i < 15; i++ {
		refSum += colMeans[i]
	}
	refMean := refSum / 10
	const threshold = 5
	var segments []mgRect
	inDifferent := false
	bgRunLen := 0
	contentStart := 0
	for i := 0; i < rectW; i++ {
		diff := colMeans[i] - refMean
		if diff < 0 {
			diff = -diff
		}
		isBg := diff <= threshold
		const segPad = 2
		if !isBg {
			if !inDifferent {
				leadX := rect.x0 + i - segPad
				if leadX < rect.x0 {
					leadX = rect.x0
				}
				contentStart = leadX - rect.x0
				inDifferent = true
			}
			bgRunLen = 0
		} else if inDifferent {
			bgRunLen++
			if bgRunLen >= 12 {
				trailX := rect.x0 + (i - bgRunLen) + segPad
				if trailX > rect.x1-1 {
					trailX = rect.x1 - 1
				}
				segments = append(segments, mgRect{rect.x0 + contentStart, rect.y0, trailX + 1, rect.y1})
				inDifferent = false
				bgRunLen = 0
			}
		}
	}
	if inDifferent {
		segments = append(segments, mgRect{rect.x0 + contentStart, rect.y0, rect.x1, rect.y1})
	}
	return segments
}

// mgFindHorizontalSegments scans a rectangle top to bottom and returns content
// (non-background) row-strip bounding boxes.
func mgFindHorizontalSegments(img image.Image, rect mgRect) []mgRect {
	rectH := rect.y1 - rect.y0
	if rectH < 16 {
		return nil
	}
	rowMeans := make([]int, rectH)
	for j := 0; j < rectH; j++ {
		y := rect.y0 + j
		sum, count := 0, 0
		for x := rect.x0; x < rect.x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			rowMeans[j] = sum / count
		}
	}
	refSum := 0
	for j := 5; j < 15; j++ {
		refSum += rowMeans[j]
	}
	refMean := refSum / 10
	const threshold = 5
	var segments []mgRect
	inDifferent := false
	bgRunLen := 0
	contentRowStart := 0
	for j := 0; j < rectH; j++ {
		diff := rowMeans[j] - refMean
		if diff < 0 {
			diff = -diff
		}
		isBg := diff <= threshold
		const segPad = 2
		if !isBg {
			if !inDifferent {
				leadY := rect.y0 + j - segPad
				if leadY < rect.y0 {
					leadY = rect.y0
				}
				contentRowStart = leadY
				inDifferent = true
			}
			bgRunLen = 0
		} else if inDifferent {
			bgRunLen++
			if bgRunLen >= 12 {
				trailY := rect.y0 + (j - bgRunLen) + segPad
				if trailY > rect.y1-1 {
					trailY = rect.y1 - 1
				}
				segments = append(segments, mgRect{rect.x0, contentRowStart, rect.x1, trailY + 1})
				inDifferent = false
				bgRunLen = 0
			}
		}
	}
	if inDifferent {
		segments = append(segments, mgRect{rect.x0, contentRowStart, rect.x1, rect.y1})
	}
	return segments
}

// mgCropAndBinarize crops rect from img and binarises it according to mode,
// returning PNG-encoded bytes scaled 3×.
//
//	0 = contrast-stretch + 2.5× boost (dark text on light bg)
//	1 = pure-white test (rank digit fill, R/G/B ≥ 240)
//	2 = contrast-stretch → invert → threshold ≤ 32 (white text on dark bg)
func mgCropAndBinarize(img image.Image, rect mgRect, mode int) ([]byte, error) {
	w := rect.x1 - rect.x0
	h := rect.y1 - rect.y0
	sub := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(sub, sub.Bounds(), img, image.Pt(rect.x0, rect.y0), draw.Src)

	if mode == 1 {
		const whiteMin = 240
		out := image.NewGray(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r16, g16, b16, _ := sub.At(x, y).RGBA()
				var v uint8
				if uint8(r16>>8) >= whiteMin && uint8(g16>>8) >= whiteMin && uint8(b16>>8) >= whiteMin {
					v = 0
				} else {
					v = 255
				}
				out.SetGray(x, y, color.Gray{Y: v})
			}
		}
		return mgEncodeUpscaled(out, w, h)
	}

	pixels := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pixels[y*w+x] = color.GrayModel.Convert(sub.At(x, y)).(color.Gray).Y
		}
	}
	minV, maxV := uint8(255), uint8(0)
	for _, g := range pixels {
		if g < minV {
			minV = g
		}
		if g > maxV {
			maxV = g
		}
	}
	span := int(maxV) - int(minV)
	if span == 0 {
		span = 1
	}
	out := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			stretched := (int(pixels[y*w+x]) - int(minV)) * 255 / span
			if mode == 0 {
				enhanced := (stretched-128)*5/2 + 128
				if enhanced < 0 {
					enhanced = 0
				} else if enhanced > 255 {
					enhanced = 255
				}
				stretched = enhanced
			} else if mode == 2 {
				inv := 255 - stretched
				if inv <= 32 {
					stretched = 0
				} else {
					stretched = 255
				}
			}
			out.SetGray(x, y, color.Gray{Y: uint8(stretched)})
		}
	}
	return mgEncodeUpscaled(out, w, h)
}

// mgEncodeUpscaled writes img upscaled 3× (nearest-neighbour) to a PNG byte slice.
func mgEncodeUpscaled(out *image.Gray, w, h int) ([]byte, error) {
	const scale = 3
	scaled := image.NewGray(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := out.GrayAt(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					scaled.SetGray(x*scale+dx, y*scale+dy, v)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── OCR helpers ─────────────────────────────────────────────────────────────

// mgRunOCR OCRs PNG bytes with the given page-segmentation mode and whitelist.
func mgRunOCR(data []byte, psm gosseract.PageSegMode, whitelist string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	if err := client.SetImageFromBytes(data); err != nil {
		return "", err
	}
	client.SetPageSegMode(psm)
	if whitelist != "" {
		client.SetVariable("tessedit_char_whitelist", whitelist) //nolint:errcheck
	}
	text, err := client.Text()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// mgOcrSegment OCRs data, optionally normalises, and returns (text, matched).
func mgOcrSegment(data []byte, psm gosseract.PageSegMode, whitelist string, pattern *regexp.Regexp, norm func(string) string) (string, bool) {
	text, err := mgRunOCR(data, psm, whitelist)
	if err != nil {
		return "", false
	}
	if norm != nil {
		text = norm(text)
	}
	return text, pattern.MatchString(text)
}

// mgReadRankDigits analyses the binarised rank badge image (mode 1) using BFS
// connected-component labelling, isolates the 1–2 digit strokes, and OCRs each
// with PSM_SINGLE_CHAR.  Returns the rank string (e.g. "42") or "" on failure.
func mgReadRankDigits(data []byte) string {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	bnd := src.Bounds()
	w, h := bnd.Dx(), bnd.Dy()

	dark := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := src.At(bnd.Min.X+x, bnd.Min.Y+y).RGBA()
			dark[y*w+x] = r>>8 < 128
		}
	}

	type ccBlob struct {
		pixels int
		bounds image.Rectangle
	}
	visited := make([]bool, w*h)
	var blobs []ccBlob
	queue := make([]int, 0, 512)
	for i, isDark := range dark {
		if !isDark || visited[i] {
			continue
		}
		ix, iy := i%w, i/w
		blob := ccBlob{bounds: image.Rect(ix, iy, ix+1, iy+1)}
		queue = queue[:0]
		queue = append(queue, i)
		visited[i] = true
		for len(queue) > 0 {
			idx := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			blob.pixels++
			cx, cy := idx%w, idx/w
			if cx < blob.bounds.Min.X {
				blob.bounds.Min.X = cx
			}
			if cy < blob.bounds.Min.Y {
				blob.bounds.Min.Y = cy
			}
			if cx+1 > blob.bounds.Max.X {
				blob.bounds.Max.X = cx + 1
			}
			if cy+1 > blob.bounds.Max.Y {
				blob.bounds.Max.Y = cy + 1
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := cx+d[0], cy+d[1]
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					continue
				}
				ni := ny*w + nx
				if dark[ni] && !visited[ni] {
					visited[ni] = true
					queue = append(queue, ni)
				}
			}
		}
		blobs = append(blobs, blob)
	}

	const minArea = 80
	var sig []ccBlob
	for _, bl := range blobs {
		if bl.pixels >= minArea {
			sig = append(sig, bl)
		}
	}
	if len(sig) == 0 {
		return ""
	}
	// Keep the 1–2 largest blobs.
	for i := 0; i < 2 && i < len(sig); i++ {
		best := i
		for j := i + 1; j < len(sig); j++ {
			if sig[j].pixels > sig[best].pixels {
				best = j
			}
		}
		sig[i], sig[best] = sig[best], sig[i]
	}
	if len(sig) > 2 {
		sig = sig[:2]
	}
	if len(sig) == 2 && sig[0].bounds.Min.X > sig[1].bounds.Min.X {
		sig[0], sig[1] = sig[1], sig[0]
	}

	const pad = 6
	var digits strings.Builder
	for _, bl := range sig {
		r := bl.bounds
		x0 := r.Min.X - pad
		if x0 < 0 {
			x0 = 0
		}
		y0 := r.Min.Y - pad
		if y0 < 0 {
			y0 = 0
		}
		x1 := r.Max.X + pad
		if x1 > w {
			x1 = w
		}
		y1 := r.Max.Y + pad
		if y1 > h {
			y1 = h
		}
		crop := image.NewGray(image.Rect(0, 0, x1-x0, y1-y0))
		for cy := y0; cy < y1; cy++ {
			for cx := x0; cx < x1; cx++ {
				val := uint8(255)
				if dark[cy*w+cx] {
					val = 0
				}
				crop.SetGray(cx-x0, cy-y0, color.Gray{Y: val})
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, crop); err != nil {
			continue
		}
		client := gosseract.NewClient()
		client.SetImageFromBytes(buf.Bytes()) //nolint:errcheck
		client.SetPageSegMode(gosseract.PSM_SINGLE_CHAR)
		client.SetVariable("tessedit_char_whitelist", "0123456789") //nolint:errcheck
		text, err := client.Text()
		client.Close()
		if err != nil {
			continue
		}
		for _, ch := range strings.TrimSpace(text) {
			if ch >= '0' && ch <= '9' {
				digits.WriteRune(ch)
				break
			}
		}
	}
	return digits.String()
}

// ─── Normalisation helpers ────────────────────────────────────────────────────

// mgNormalizeName repairs OCR artefacts in alliance-tag + player-name strings.
func mgNormalizeName(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return s
	}
	if s[0] != '[' {
		s = "[" + s
	}
	s = regexp.MustCompile(`\[([A-Za-z0-9]{1,4})[ |]+\]?`).ReplaceAllString(s, "[$1]")
	return s
}

// mgNormalizeDamage reconstructs "Total Damage: X.XXG/M" from common OCR
// misreadings.  Handles dropped decimals (XXYY → XX.YY) and '.' misread as '4'.
func mgNormalizeDamage(raw string) string {
	re := regexp.MustCompile(`(\d+)(?:[.:](\d{1,2}))?[^0-9GM]*([GM])`)
	trailingNoise := regexp.MustCompile(`[^0-9GM]+$`)

	apply := func(s string) string {
		m := re.FindStringSubmatch(s)
		if m == nil {
			return ""
		}
		intPart, decPart, unit := m[1], m[2], strings.ToUpper(m[3])
		if decPart != "" {
			return fmt.Sprintf("Total Damage: %s.%s%s", intPart, decPart, unit)
		}
		// Reject implausibly long integer parts — they are OCR noise, not real values.
		// Real values: 5G, 23G, 962M (1–3 digits). Recovery handles 4–5 digit cases below.
		if len(intPart) > 5 {
			return ""
		}
		switch len(intPart) {
		case 4:
			return fmt.Sprintf("Total Damage: %s.%s%s", intPart[:2], intPart[2:], unit)
		case 5:
			if intPart[2] == '4' {
				return fmt.Sprintf("Total Damage: %s.%s%s", intPart[:2], intPart[3:], unit)
			}
		}
		return fmt.Sprintf("Total Damage: %s%s", intPart, unit)
	}

	if result := apply(raw); result != "" {
		return result
	}
	cleaned := trailingNoise.ReplaceAllString(raw, "")
	if result := apply(cleaned); result != "" {
		return result
	}
	sixToG := regexp.MustCompile(`6(\s*)$`)
	if sixToG.MatchString(cleaned) {
		if result := apply(sixToG.ReplaceAllString(cleaned, "G$1")); result != "" {
			return result
		}
	}
	return raw
}

// mgParseDamageInt converts a "Total Damage: X.XXG/M" string to int64 (raw units).
func mgParseDamageInt(s string) int64 {
	m := regexp.MustCompile(`Total Damage:\s*(\d+)(?:\.(\d{1,2}))?([GM])`).FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	intPart, _ := strconv.ParseInt(m[1], 10, 64)
	decStr := m[2]
	unit := strings.ToUpper(m[3])
	var multiplier int64
	if unit == "G" {
		multiplier = 1_000_000_000
	} else {
		multiplier = 1_000_000
	}
	val := intPart * multiplier
	if decStr != "" {
		// Pad or trim to 2 decimal places.
		for len(decStr) < 2 {
			decStr += "0"
		}
		if len(decStr) > 2 {
			decStr = decStr[:2]
		}
		frac, _ := strconv.ParseInt(decStr, 10, 64)
		val += frac * (multiplier / 100)
	}
	return val
}

// mgFormatDamageStr converts an int64 damage value (raw units) to a display string like "2.37G" or "498.13M".
func mgFormatDamageStr(d int64) string {
	if d >= 1_000_000_000 {
		return fmt.Sprintf("%.2fG", float64(d)/1e9)
	}
	return fmt.Sprintf("%.2fM", float64(d)/1e6)
}

// mgParseDamageStr extracts just the "X.XXG" portion from a full "Total Damage: X.XXG" string.
func mgParseDamageStr(s string) string {
	m := regexp.MustCompile(`Total Damage:\s*(.+)`).FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return s
	}
	return strings.TrimSpace(m[1])
}

// ─── Rank sequence reconstruction ────────────────────────────────────────────

type mgMemberRow struct {
	rankStr string // "OK \"7\"" / "FAIL \"...\"" / "CORR \"9\"" / "FIXD …"
	name    string
	nameOK  bool
	dmgStr  string
	dmgOK   bool
	cropY0  float64 // top of row relative to full image (0.0–1.0)
	cropY1  float64 // bottom of row relative to full image
}

// mgReconstructSequence fills FAIL ranks and corrects misreads using the
// fact that ranks within one screenshot are consecutive with step 1.
func mgReconstructSequence(rows []mgMemberRow) {
	if len(rows) == 0 {
		return
	}
	rankVal := func(s string) int {
		if !strings.HasPrefix(strings.TrimSpace(s), "OK") {
			return -1
		}
		m := regexp.MustCompile(`"(\d+)"`).FindStringSubmatch(s)
		if m == nil {
			return -1
		}
		v, _ := strconv.Atoi(m[1])
		return v
	}
	votes := map[int]int{}
	for j := range rows {
		if v := rankVal(rows[j].rankStr); v >= 0 {
			votes[v-j]++
		}
	}
	if len(votes) == 0 {
		return
	}
	bestStart, bestCount := 0, 0
	for s, c := range votes {
		if c > bestCount || (c == bestCount && s > bestStart) {
			bestStart, bestCount = s, c
		}
	}
	for j := range rows {
		expected := bestStart + j
		cur := rankVal(rows[j].rankStr)
		if cur == expected {
			continue
		}
		es := strconv.Itoa(expected)
		if cur < 0 {
			rows[j].rankStr = fmt.Sprintf("CORR %q", es)
		} else {
			rows[j].rankStr = fmt.Sprintf("FIXD %q (was %q)", es, strconv.Itoa(cur))
		}
	}
}

// ─── Colored region detection ─────────────────────────────────────────────────

func mgIsRowNonWhite(img image.Image, y, width int) bool {
	const whiteThreshold = 240
	nonWhite := 0
	step := 5
	for x := 0; x < width; x += step {
		r, g, b, _ := img.At(x, y).RGBA()
		if uint8(r>>8) < whiteThreshold || uint8(g>>8) < whiteThreshold || uint8(b>>8) < whiteThreshold {
			nonWhite++
		}
	}
	return float64(nonWhite)/float64(width/step) > 0.3
}

func mgFindColumnBounds(img image.Image, y0, y1, width int) (int, int) {
	colMean := make([]int, width)
	for x := 0; x < width; x++ {
		sum, count := 0, 0
		for y := y0; y < y1; y += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMean[x] = sum / count
		}
	}
	const colorThreshold = 230
	const margin = 3
	leftMost := margin
	for x := margin; x < width-margin; x++ {
		if colMean[x] < colorThreshold {
			leftMost = x
			break
		}
	}
	rightMost := width - margin - 1
	for x := width - margin - 1; x >= margin; x-- {
		if colMean[x] < colorThreshold {
			rightMost = x
			break
		}
	}
	return leftMost, rightMost + 1
}

func mgFindColoredRegions(img image.Image, width, height int) []mgRect {
	var rects []mgRect
	inRegion := false
	regionStart := 0
	for y := 0; y < height; y++ {
		isNonWhite := mgIsRowNonWhite(img, y, width)
		if isNonWhite && !inRegion {
			regionStart = y
			inRegion = true
		} else if !isNonWhite && inRegion {
			if y-regionStart > 20 {
				x0, x1 := mgFindColumnBounds(img, regionStart, y, width)
				rects = append(rects, mgRect{x0, regionStart, x1, y})
			}
			inRegion = false
		}
	}
	if inRegion && height-regionStart > 20 {
		x0, x1 := mgFindColumnBounds(img, regionStart, height, width)
		rects = append(rects, mgRect{x0, regionStart, x1, height})
	}
	return rects
}

// ─── Main processing function ─────────────────────────────────────────────────

// mgProcessImage decodes one Marshal Guard screenshot from PNG/JPEG bytes and
// returns the structured OCR results.
func mgProcessImage(imageData []byte) (*MGImgResult, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Crop to the dialog bounding box.
	dialogX0, dialogY0, dialogX1, dialogY1 := mgFindDialogBounds(img, width, height)
	cropW := dialogX1 - dialogX0
	cropH := dialogY1 - dialogY0
	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(dialogX0, dialogY0), draw.Src)

	rectangles := mgFindColoredRegions(cropped, cropW, cropH)

	var rows []mgMemberRow
	result := &MGImgResult{}

	for _, rect := range rectangles {
		vertSegs := mgFindVerticalSegments(cropped, rect)
		hSegs := make([][]mgRect, len(vertSegs))
		for si, seg := range vertSegs {
			hSegs[si] = mgFindHorizontalSegments(cropped, seg)
		}
		hLens := make([]int, len(hSegs))
		for i, h := range hSegs {
			hLens[i] = len(h)
		}
		log.Printf("mg_ocr: rect %v vsegs=%d hSegs=%v", rect, len(vertSegs), hLens)

		// ── Top player card ──
		// Layout A (original mg_segment): 3 vsegs, hSegs[2]>=1, hSegs[1]>=2  → name in hSegs[2][0]
		//   damage comes from a SEPARATE rect: 3 vsegs, hSegs[1]==1, hSegs[2]==1 (handled below).
		// Layout B (4-vseg): vseg[3] is a combined text block with both name and damage.
		// Layout C (2-vseg): hSegs[0]>=2, hSegs[1]==1 → text area is hSegs[1][0].
		var topNameRect, topDmgRect mgRect
		topLayoutB := false
		topDetected := false
		if len(vertSegs) == 3 && len(hSegs[1]) >= 2 && len(hSegs[2]) >= 1 {
			topNameRect = hSegs[2][0]
			topDetected = true
		} else if len(vertSegs) == 4 && len(hSegs) >= 4 && len(hSegs[3]) >= 1 {
			// vseg[3] is the text area containing name and damage as a block.
			topNameRect = hSegs[3][0]
			topDmgRect = hSegs[3][0]
			topLayoutB = true
			topDetected = true
		} else if len(vertSegs) == 2 && len(hSegs) >= 2 && len(hSegs[0]) >= 2 && len(hSegs[1]) == 1 {
			// Layout C: 2 vsegs — left side is avatar (multi-band), right side is text block.
			topNameRect = hSegs[1][0]
			topDmgRect = hSegs[1][0]
			topLayoutB = true
			topDetected = true
		}
		if topDetected && result.TopPlayerName == "" {
			data, err := mgCropAndBinarize(cropped, topNameRect, 0)
			if err != nil {
				log.Printf("mg_ocr: top card binarize: %v", err)
			} else {
				// Use SINGLE_BLOCK with no whitelist so name characters are captured.
				raw, _ := mgRunOCR(data, gosseract.PSM_SINGLE_BLOCK, "")
				log.Printf("mg_ocr: top card raw OCR: %q", raw)
				for _, line := range strings.Split(raw, "\n") {
					line = strings.TrimSpace(line)
					if idx := strings.Index(line, "["); idx >= 0 {
						candidate := strings.TrimSpace(line[idx:])
						norm := mgNormalizeName(candidate)
						if mgNameRe.MatchString(norm) {
							result.TopPlayerName = norm
							log.Printf("mg_ocr: top player name: %q", norm)
							break
						}
					}
				}
				// Layout B: also try to extract damage from the same block.
				if topLayoutB && result.TopPlayerDmgStr == "" {
					for _, line := range strings.Split(raw, "\n") {
						line = strings.TrimSpace(line)
						norm := mgNormalizeDamage(line)
						if mgDamageRe.MatchString(norm) {
							result.TopPlayerDmgStr = mgParseDamageStr(norm)
							result.TopPlayerDmgInt = mgParseDamageInt(norm)
							log.Printf("mg_ocr: top damage (layout B from block): %q", norm)
							break
						}
					}
				}
			}
		}
		// Top player damage from layout B/C: try all three binarise modes and keep the first
		// non-zero result (mode 2 finds the text but can misread digits; mode 0/1 may do better).
		if topDetected && topLayoutB && (topDmgRect != mgRect{}) && result.TopPlayerDmgStr == "" {
			for _, bmode := range []int{2, 0, 1} {
				data, err := mgCropAndBinarize(cropped, topDmgRect, bmode)
				if err != nil {
					continue
				}
				dmg, ok := mgOcrSegment(data, gosseract.PSM_SINGLE_LINE, "TotalDamge :0123456789.,GM", mgDamageRe, mgNormalizeDamage)
				log.Printf("mg_ocr: top damage OCR mode=%d ok=%v %q", bmode, ok, dmg)
				if ok && mgParseDamageInt(dmg) > 0 {
					result.TopPlayerDmgStr = mgParseDamageStr(dmg)
					result.TopPlayerDmgInt = mgParseDamageInt(dmg)
					break
				}
			}
			if result.TopPlayerDmgStr == "" {
				log.Printf("mg_ocr: top damage could not be extracted for layout B/C")
			}
		}

		// ── Top player damage row (layout A): 3 vsegs, hSegs[1]==1, hSegs[2]==1 ──
		if len(vertSegs) == 3 && len(hSegs[1]) == 1 && len(hSegs[2]) == 1 {
			data, err := mgCropAndBinarize(cropped, hSegs[2][0], 2)
			if err != nil {
				log.Printf("mg_ocr: top damage binarize: %v", err)
				continue
			}
			dmg, ok := mgOcrSegment(data, gosseract.PSM_SINGLE_LINE, "TotalDamge :0123456789.,GM", mgDamageRe, mgNormalizeDamage)
			log.Printf("mg_ocr: top damage OCR (layout A) ok=%v %q", ok, dmg)
			if ok && result.TopPlayerDmgStr == "" {
				result.TopPlayerDmgStr = mgParseDamageStr(dmg)
				result.TopPlayerDmgInt = mgParseDamageInt(dmg)
			}
		}

		// ── Member row: 5 vsegs, hSegs[1]==3, hSegs[3]==4 ──
		if len(vertSegs) == 5 && len(hSegs[1]) == 3 && len(hSegs[3]) == 4 {
			rankData, err := mgCropAndBinarize(cropped, hSegs[1][1], 1)
			if err != nil {
				continue
			}
			nameData, err := mgCropAndBinarize(cropped, hSegs[3][1], 0)
			if err != nil {
				continue
			}
			dmgData, err := mgCropAndBinarize(cropped, hSegs[3][2], 2)
			if err != nil {
				continue
			}

			rankStr := mgReadRankDigits(rankData)
			var rankLabel string
			if mgRankRe.MatchString(rankStr) {
				rankLabel = fmt.Sprintf("OK %q", rankStr)
			} else {
				rankLabel = fmt.Sprintf("FAIL %q", rankStr)
			}

			name, nameOK := mgOcrSegment(nameData, gosseract.PSM_SINGLE_LINE,
				"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789[] ",
				mgNameRe, mgNormalizeName)
			dmg, dmgOK := mgOcrSegment(dmgData, gosseract.PSM_SINGLE_LINE,
				"TotalDamge :0123456789.,GM", mgDamageRe, mgNormalizeDamage)

			rows = append(rows, mgMemberRow{
				rankStr: rankLabel,
				name:    name,
				nameOK:  nameOK,
				dmgStr:  dmg,
				dmgOK:   dmgOK,
				cropY0:  float64(rect.y0+dialogY0) / float64(height),
				cropY1:  float64(rect.y1+dialogY0) / float64(height),
			})
		}

		// ── Datetime: 1 vseg, hSegs[0]>=2 (originally ==3, some layouts ==2) ──
		if len(vertSegs) == 1 && len(hSegs[0]) >= 2 && result.EventDate == "" {
			dateRe := regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
			for _, hseg := range hSegs[0] {
				data, err := mgCropAndBinarize(cropped, hseg, 0)
				if err != nil {
					continue
				}
				rawDt, _ := mgRunOCR(data, gosseract.PSM_SINGLE_LINE, "")
				log.Printf("mg_ocr: datetime hseg %v raw OCR: %q", hseg, rawDt)
				if m := dateRe.FindStringSubmatch(rawDt); m != nil {
					y, _ := strconv.Atoi(m[1])
					mo, _ := strconv.Atoi(m[2])
					d, _ := strconv.Atoi(m[3])
					result.EventDate = fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
					log.Printf("mg_ocr: event date: %q", result.EventDate)
					break
				}
			}
		}
	}

	// Reconstruct rank sequence.
	mgReconstructSequence(rows)

	// Convert rows to MGImgResult.Members.
	rankExtract := regexp.MustCompile(`"(\d+)"`)
	for _, row := range rows {
		m := rankExtract.FindStringSubmatch(row.rankStr)
		if m == nil {
			log.Printf("mg_ocr: could not extract rank from %q", row.rankStr)
			continue
		}
		rank, _ := strconv.Atoi(m[1])
		fixed := strings.HasPrefix(row.rankStr, "CORR") || strings.HasPrefix(row.rankStr, "FIXD")
		dmgStr := mgParseDamageStr(row.dmgStr)
		dmgInt := mgParseDamageInt(row.dmgStr)
		result.Members = append(result.Members, mgMemberOCR{
			Rank:      rank,
			Name:      row.name,
			NameOK:    row.nameOK,
			DamageStr: dmgStr,
			DamageInt: dmgInt,
			DamageOK:  row.dmgOK,
			RankFixed: fixed,
			CropY0:    row.cropY0,
			CropY1:    row.cropY1,
		})
	}

	sort.Slice(result.Members, func(i, j int) bool {
		return result.Members[i].Rank < result.Members[j].Rank
	})

	return result, nil
}
