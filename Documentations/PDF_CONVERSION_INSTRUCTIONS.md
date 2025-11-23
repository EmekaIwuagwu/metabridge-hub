# Whitepaper PDF Conversion Instructions

## Quick Start (Recommended Methods)

### Method 1: Using Pandoc (Best Quality)

```bash
# Install pandoc (one-time setup)
# macOS:
brew install pandoc basictex

# Ubuntu/Debian:
sudo apt-get install pandoc texlive-latex-base texlive-fonts-recommended texlive-latex-extra

# Windows:
# Download from https://pandoc.org/installing.html

# Convert to PDF
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V documentclass=report \
  -V papersize=letter
```

### Method 2: Using Markdown to PDF (Node.js)

```bash
# Install globally
npm install -g md-to-pdf

# Convert
md-to-pdf WHITEPAPER.md
```

### Method 3: Using Grip (GitHub-style)

```bash
# Install
pip install grip

# Convert (opens in browser, then Print to PDF)
grip WHITEPAPER.md
# Open http://localhost:6419 in browser
# File → Print → Save as PDF
```

### Method 4: Using VS Code Extension

1. Install "Markdown PDF" extension in VS Code
2. Open WHITEPAPER.md
3. Press Ctrl+Shift+P (Cmd+Shift+P on Mac)
4. Type "Markdown PDF: Export (pdf)"
5. Select PDF format

### Method 5: Using Online Converters

**Recommended Online Tools**:

1. **Dillinger** (https://dillinger.io/)
   - Paste markdown content
   - Export as PDF

2. **Markdown to PDF** (https://www.markdowntopdf.com/)
   - Upload WHITEPAPER.md
   - Download PDF

3. **CloudConvert** (https://cloudconvert.com/md-to-pdf)
   - Upload WHITEPAPER.md
   - Convert and download

## Advanced Pandoc Options

### Professional Document with Cover Page

```bash
# Create cover page (cover.tex)
cat > cover.tex << 'EOF'
\begin{titlepage}
\centering
\vspace*{2cm}
{\Huge\bfseries Articium\par}
\vspace{0.5cm}
{\LARGE Universal Cross-Chain Bridge Protocol\par}
\vspace{1cm}
{\Large Technical Whitepaper\par}
\vspace{0.5cm}
{\large Version 1.0\par}
\vspace{2cm}
{\large November 2025\par}
\vfill
{\large Production-Ready Implementation\par}
\end{titlepage}
EOF

# Convert with cover
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V documentclass=report \
  -V papersize=letter \
  -V colorlinks=true \
  -V linkcolor=blue \
  -V urlcolor=blue \
  -V toccolor=black \
  --include-before-body=cover.tex \
  --metadata title="Articium" \
  --metadata author="Articium Team" \
  --metadata date="November 2025"
```

### With Custom Styling

```bash
# Create custom template (custom.tex)
cat > custom.tex << 'EOF'
\usepackage{fancyhdr}
\pagestyle{fancy}
\fancyhead[L]{Articium}
\fancyhead[R]{Technical Whitepaper v1.0}
\fancyfoot[C]{\thepage}
EOF

# Convert with custom styling
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V documentclass=report \
  --include-in-header=custom.tex
```

## Troubleshooting

### Issue: "pandoc: xelatex not found"

**Solution**: Install LaTeX distribution
```bash
# macOS
brew install basictex
sudo tlmgr update --self
sudo tlmgr install collection-fontsrecommended

# Ubuntu/Debian
sudo apt-get install texlive-xetex texlive-latex-extra

# Windows
# Download MiKTeX: https://miktex.org/download
```

### Issue: "Package X not found"

**Solution**: Install missing LaTeX packages
```bash
# macOS/Linux
sudo tlmgr install <package-name>

# Or install full LaTeX distribution
# macOS:
brew install texlive

# Ubuntu:
sudo apt-get install texlive-full
```

### Issue: "Too many levels of heading"

**Solution**: Reduce heading depth in conversion
```bash
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --toc-depth=2 \
  --number-sections
```

## Docker-Based Conversion (No Local Installation)

```bash
# Use Docker pandoc image
docker run --rm -v "$(pwd):/data" \
  pandoc/latex:latest \
  WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --number-sections \
  -V geometry:margin=1in
```

## GitHub Actions Workflow (Automated)

Create `.github/workflows/pdf-generation.yml`:

```yaml
name: Generate Whitepaper PDF

on:
  push:
    paths:
      - 'WHITEPAPER.md'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Pandoc
        run: |
          sudo apt-get update
          sudo apt-get install -y pandoc texlive-xetex texlive-latex-extra

      - name: Generate PDF
        run: |
          pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
            --pdf-engine=xelatex \
            --toc \
            --number-sections \
            -V geometry:margin=1in

      - name: Upload PDF
        uses: actions/upload-artifact@v3
        with:
          name: whitepaper-pdf
          path: WHITEPAPER.pdf
```

## Quality Recommendations

### For Investor Presentations

**Use**:
- Method 1 (Pandoc with XeLaTeX) - Highest quality
- Professional fonts (Times New Roman, Georgia)
- Table of contents with page numbers
- Page numbers in footer
- Section numbering
- Hyperlinked table of contents
- 1-inch margins
- 11pt or 12pt font

**Command**:
```bash
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=12pt \
  -V documentclass=report \
  -V papersize=letter \
  -V colorlinks=true \
  -V mainfont="Times New Roman" \
  --metadata title="Articium - Technical Whitepaper" \
  --metadata author="Articium Team" \
  --metadata date="November 2025"
```

### For Grant Applications

**Use**:
- Clear, professional formatting
- Prominent header with project name
- Page numbers
- Section references
- Technical accuracy preserved

### For Web Distribution

**Generate Multiple Formats**:
```bash
# PDF for download
pandoc WHITEPAPER.md -o WHITEPAPER.pdf --pdf-engine=xelatex --toc

# HTML for web viewing
pandoc WHITEPAPER.md -o WHITEPAPER.html --toc --standalone --css=style.css

# EPUB for ebook readers
pandoc WHITEPAPER.md -o WHITEPAPER.epub --toc
```

## Pre-Conversion Checklist

- [ ] Review markdown for formatting issues
- [ ] Ensure all links are working
- [ ] Check image references (if any)
- [ ] Verify table formatting
- [ ] Test conversion with sample pages first
- [ ] Review generated PDF for accuracy
- [ ] Check page breaks and section breaks
- [ ] Verify table of contents accuracy
- [ ] Test all hyperlinks in PDF

## Post-Conversion Quality Check

1. **Visual Inspection**:
   - Open PDF in Adobe Reader or Preview
   - Check first 3 pages for formatting
   - Verify table of contents
   - Check all section headers

2. **Content Verification**:
   - Verify page count (should be 60-80 pages)
   - Check no content is cut off
   - Verify tables are readable
   - Check code blocks are formatted

3. **Hyperlinks**:
   - Click table of contents links
   - Verify external URLs work
   - Check internal references

4. **File Size**:
   - Typical size: 1-5 MB
   - If > 10 MB, optimize images or compress

## Quick Reference Commands

```bash
# Basic conversion
pandoc WHITEPAPER.md -o WHITEPAPER.pdf

# With table of contents
pandoc WHITEPAPER.md -o WHITEPAPER.pdf --toc

# Professional quality
pandoc WHITEPAPER.md -o WHITEPAPER.pdf \
  --pdf-engine=xelatex \
  --toc \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=12pt

# Using Docker (no installation needed)
docker run --rm -v "$(pwd):/data" pandoc/latex \
  WHITEPAPER.md -o WHITEPAPER.pdf --toc

# Online (open browser, print to PDF)
python3 -m http.server 8000
# Navigate to http://localhost:8000/WHITEPAPER.md
# Use browser extension like "Markdown Viewer" then Print to PDF
```

## Support & Resources

**Pandoc Documentation**: https://pandoc.org/MANUAL.html
**LaTeX Installation**: https://www.latex-project.org/get/
**Markdown Guide**: https://www.markdownguide.org/
**PDF Optimization**: https://www.adobe.com/acrobat/

---

**Note**: For the highest quality PDF suitable for investor presentations and grant applications, we recommend **Method 1 (Pandoc with XeLaTeX)**.
