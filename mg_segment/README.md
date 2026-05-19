# Marshal Guard Screenshot Segmentation

Pre-processing tool to segment Marshal Guard event screenshots into analyzable components.

## What it does

Segments each screenshot into:

1. **MVP Section** (top ~30%): Contains the MVP player with trophy building
2. **Member Rows** (middle): Individual ranking entries, each containing:
   - Rank number (left column)
   - Avatar image (square)
   - Alliance tag + player name (top row of text)
   - "Total Damage: X" (bottom row of text)
3. **Date Section** (bottom ~10%): Event timestamp

## Output Files

For each input image `screenshot.png`, creates:

- `screenshot_annotated.png` - Full image with colored borders showing detected regions
- `screenshot_mvp.png` - MVP section crop
- `screenshot_date.png` - Date section crop
- `screenshot_row_00_full.png` - Complete row 0
- `screenshot_row_00_rank.png` - Rank number only
- `screenshot_row_00_avatar.png` - Avatar image only
- `screenshot_row_00_text_top.png` - Alliance + name
- `screenshot_row_00_text_bottom.png` - Damage value
- (repeats for each detected row)

## Usage

### With Docker

```bash
# Build
cd mg_segment
docker build -t mg_segment .

# Run (Windows)
docker run --rm -v C:\Users\verve\Downloads:/input -v f:\Projects\LastWar\mg_output:/output mg_segment

# Run (Linux/Mac)
docker run --rm -v ~/Downloads:/input -v ./mg_output:/output mg_segment
```

### Direct (requires Go 1.21+)

```bash
cd mg_segment
go run main.go C:\Users\verve\Downloads .\output
```

## Algorithm

1. **Convert to grayscale** for edge detection
2. **Detect MVP section**: Find horizontal separator using line uniformity score
3. **Detect date section**: Bottom 10% of image
4. **Find member rows**: Scan for separator lines between entries
5. **Segment each row**:
   - Rank: First 12.5% of width
   - Avatar: Square region after rank
   - Text: Remaining width, split top/bottom for name and damage

Uses Sobel edge detection to distinguish avatar (high complexity) from text background (low complexity).
