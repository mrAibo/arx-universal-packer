from pathlib import Path

replacements = {
    "arx-go/archive_selection.go": (
        "extracted := extractSelected(source, nil, temporary)",
        "extracted := extract(source, temporary)",
    ),
    "arx-go/archive_edit.go": (
        "if extracted := extractSelected(archive, nil, staging); extracted.Err != nil {",
        "if extracted := extract(archive, staging); extracted.Err != nil {",
    ),
    "arx-go/main.go": (
        "return extractSelected(archivePath, nil, passive.path)",
        "return extract(archivePath, passive.path)",
    ),
}

for name, (old, new) in replacements.items():
    path = Path(name)
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{name}: expected one match, found {count}")
    path.write_text(text.replace(old, new, 1))
