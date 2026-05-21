import sys, re

content = open('main.go', 'r', encoding='utf-8').read()

# ─── Patch 1: row builder — add SourceFileIdx, CropY0Pct, CropY1Pct ───────────
old1 = (
    '\t\t\t\t\tm := acc.members[rank]\n'
    '\t\t\t\t\t// Parse alliance tag and plain name from "[TAG]PlayerName".\n'
    '\t\t\t\t\tallianceTag, nameOnly := parsePlayerTag(m.Name, knownTag)\n'
    '\t\t\t\t\trow := MGV2PreviewRow{\n'
    '\t\t\t\t\t\tRank:        rank,\n'
    '\t\t\t\t\t\tName:        nameOnly,\n'
    '\t\t\t\t\t\tAllianceTag: allianceTag,\n'
    '\t\t\t\t\t\tNameOK:      m.NameOK,\n'
    '\t\t\t\t\t\tDamageStr:   m.DamageStr,\n'
    '\t\t\t\t\t\tDamage:      m.DamageInt,\n'
    '\t\t\t\t\t\tDamageOK:    m.DamageOK,\n'
    '\t\t\t\t\t\tRankFixed:   m.RankFixed,\n'
    '\t\t\t\t\t}'
)
new1 = (
    '\t\t\t\t\tm := acc.members[rank]\n'
    '\t\t\t\t\t// Parse alliance tag and plain name from "[TAG]PlayerName".\n'
    '\t\t\t\t\tallianceTag, nameOnly := parsePlayerTag(m.Name, knownTag)\n'
    '\t\t\t\t\tsrcIdx := m.FileIdx\n'
    '\t\t\t\t\trow := MGV2PreviewRow{\n'
    '\t\t\t\t\t\tRank:          rank,\n'
    '\t\t\t\t\t\tName:          nameOnly,\n'
    '\t\t\t\t\t\tAllianceTag:   allianceTag,\n'
    '\t\t\t\t\t\tNameOK:        m.NameOK,\n'
    '\t\t\t\t\t\tDamageStr:     m.DamageStr,\n'
    '\t\t\t\t\t\tDamage:        m.DamageInt,\n'
    '\t\t\t\t\t\tDamageOK:      m.DamageOK,\n'
    '\t\t\t\t\t\tRankFixed:     m.RankFixed,\n'
    '\t\t\t\t\t\tSourceFileIdx: &srcIdx,\n'
    '\t\t\t\t\t\tCropY0Pct:     m.CropY0,\n'
    '\t\t\t\t\t\tCropY1Pct:     m.CropY1,\n'
    '\t\t\t\t\t}'
)

# ─── Patch 2: OCR-normalization helpers + second pass in matchMGParticipant ────
# Insert before the existing matchMGParticipant function and add second pass inside it.

helpers = '''
// mgOcrNormForCompare maps visually-similar characters to a canonical form for
// name-matching only, so OCR confusions (O\u21940, l/I\u21941) do not prevent a fuzzy match.
func mgOcrNormForCompare(s string) string {
\ts = strings.ToLower(s)
\ts = strings.NewReplacer("o", "0", "l", "1", "i", "1").Replace(s)
\treturn s
}

// mgSimilarityNorm computes the Levenshtein-based similarity (0\u2013100) between two
// already-normalised strings.
func mgSimilarityNorm(n1, n2 string) int {
\tif n1 == n2 {
\t\treturn 100
\t}
\tdist := levenshteinDistance(n1, n2)
\tmaxLen := len(n1)
\tif len(n2) > maxLen {
\t\tmaxLen = len(n2)
\t}
\tif maxLen == 0 {
\t\treturn 0
\t}
\treturn (maxLen - dist) * 100 / maxLen
}

'''

# Locate the matchMGParticipant function to insert helpers before it
insert_before = 'func matchMGParticipant(p *MGOCRParticipant, members []Member) {'
if helpers.strip()[:30] not in content:
    content = content.replace(insert_before, helpers + insert_before, 1)
    print('[patch 2a] OCR helper functions inserted')
else:
    print('[patch 2a] helpers already present, skipped')

# Add second pass to matchMGParticipant — find the closing of the first bestScore block
old2 = (
    '\tif bestScore >= 70 && bestIdx >= 0 {\n'
    '\t\tp.MemberID = &members[bestIdx].ID\n'
    '\t\tp.MemberName = members[bestIdx].Name\n'
    '\t}\n'
    '}\n'
    '\n'
    '// extractMGByMask'
)
new2 = (
    '\tif bestScore >= 70 && bestIdx >= 0 {\n'
    '\t\tp.MemberID = &members[bestIdx].ID\n'
    '\t\tp.MemberName = members[bestIdx].Name\n'
    '\t\treturn\n'
    '\t}\n'
    '\n'
    '\t// Second pass: OCR character normalisation (O\u21940, l/I\u21941, try without\n'
    '\t// spurious leading character added by OCR e.g. "J" before "KM011").\n'
    '\tocrNorm := mgOcrNormForCompare(name)\n'
    '\tbestScore = 0\n'
    '\tbestIdx = -1\n'
    '\tfor i, m := range members {\n'
    '\t\tdbNorm := mgOcrNormForCompare(m.Name)\n'
    '\t\tsim := mgSimilarityNorm(ocrNorm, dbNorm)\n'
    '\t\t// Also try without the first character (spurious leading OCR char).\n'
    '\t\tif len(ocrNorm) > 3 {\n'
    '\t\t\tif s2 := mgSimilarityNorm(ocrNorm[1:], dbNorm); s2 > sim {\n'
    '\t\t\t\tsim = s2\n'
    '\t\t\t}\n'
    '\t\t}\n'
    '\t\tif m.Nickname != nil && *m.Nickname != "" {\n'
    '\t\t\tnickNorm := mgOcrNormForCompare(*m.Nickname)\n'
    '\t\t\tif s := mgSimilarityNorm(ocrNorm, nickNorm); s > sim {\n'
    '\t\t\t\tsim = s\n'
    '\t\t\t}\n'
    '\t\t}\n'
    '\t\tif sim > bestScore {\n'
    '\t\t\tbestScore = sim\n'
    '\t\t\tbestIdx = i\n'
    '\t\t}\n'
    '\t}\n'
    '\tif bestScore >= 70 && bestIdx >= 0 {\n'
    '\t\tp.MemberID = &members[bestIdx].ID\n'
    '\t\tp.MemberName = members[bestIdx].Name\n'
    '\t}\n'
    '}\n'
    '\n'
    '// extractMGByMask'
)

if '// Second pass: OCR character' not in content:
    if old2 in content:
        content = content.replace(old2, new2, 1)
        print('[patch 2b] second pass added to matchMGParticipant')
    else:
        print('[patch 2b] COULD NOT FIND closing block of matchMGParticipant — manual edit required')
        # Dump context for debugging
        idx = content.find('if bestScore >= 70 && bestIdx >= 0')
        print(repr(content[idx:idx+300]))
else:
    print('[patch 2b] second pass already present, skipped')

if old1 in content:
    content = content.replace(old1, new1, 1)
    print('[patch 1] row builder updated')
else:
    print('[patch 1] COULD NOT FIND row builder — checking...')
    idx = content.find('Rank:        rank,')
    print(repr(content[idx-200:idx+400]))

open('main.go', 'w', encoding='utf-8').write(content)
print('Done.')
