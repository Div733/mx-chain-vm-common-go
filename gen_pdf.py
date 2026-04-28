import markdown
from weasyprint import HTML

md_path = "SECURITY_FINDINGS_REPORT_mx-chain-vm-common-go.md"
pdf_path = "SECURITY_FINDINGS_REPORT_mx-chain-vm-common-go.pdf"

with open(md_path, "r") as f:
    md_text = f.read()

body_html = markdown.markdown(
    md_text,
    extensions=["tables", "fenced_code", "nl2br"]
)

CSS = """
@page {
  size: A4 portrait;
  margin: 15mm 12mm 15mm 12mm;
}
* { box-sizing: border-box; }
body {
  font-family: 'Arimo', 'Liberation Sans', Arial, sans-serif;
  font-size: 10pt;
  color: #1A1A1A;
  background: #fff;
  margin: 0;
  line-height: 1.4;
}
h1 {
  font-size: 17pt;
  font-weight: bold;
  color: #1A1A1A;
  margin: 0 0 12px 0;
  page-break-after: avoid;
}
h2 {
  font-size: 13pt;
  font-weight: bold;
  color: #111;
  margin: 16px 0 8px 0;
  page-break-after: avoid;
}
h3 {
  font-size: 11pt;
  font-weight: bold;
  color: #222;
  margin: 12px 0 6px 0;
  page-break-after: avoid;
}
p  { margin: 6px 0; }
ul, ol { margin: 6px 0 6px 20px; padding: 0; }
li { margin: 3px 0; }
pre {
  background: #F0F0F0;
  border: 1px solid #DCDCDC;
  padding: 8px 10px;
  font-family: 'Cousine', 'Liberation Mono', 'Courier New', monospace;
  font-size: 7.5pt;
  white-space: pre-wrap;
  word-wrap: break-word;
  page-break-inside: avoid;
  color: #1A1A1A;
}
code {
  font-family: 'Cousine', 'Liberation Mono', 'Courier New', monospace;
  font-size: 7.5pt;
  background: #F0F0F0;
  padding: 1px 3px;
  color: #1A1A1A;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 7pt;
  margin: 10px 0;
  page-break-inside: auto;
  table-layout: fixed;
  word-wrap: break-word;
}
thead { display: table-header-group; }
tr {
  page-break-inside: avoid;
  page-break-after: auto;
}
th, th * {
  background: #1A1A1A;
  color: #ffffff;
  padding: 4px 4px;
  text-align: left;
  font-weight: bold;
  border: 1px solid #1A1A1A;
  font-family: 'Arimo', 'Liberation Sans', Arial, sans-serif;
  font-size: 7pt;
  word-wrap: break-word;
  overflow-wrap: break-word;
}
th p, th span, th strong, th em, th code {
  color: #ffffff;
  background: transparent;
}
td {
  padding: 4px 4px;
  border: 1px solid #1A1A1A;
  vertical-align: top;
  word-wrap: break-word;
  overflow-wrap: break-word;
  color: #1A1A1A;
  font-size: 7pt;
  line-height: 1.3;
}
tr:nth-child(even) td { background: #F8F8F8; }
hr {
  border: none;
  border-top: 0.8px solid #DCDCDC;
  margin: 12px 0;
}
strong { font-weight: bold; color: #000; }
em { font-style: italic; }
"""

html = f"""<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8"/>
<style>{CSS}</style>
</head>
<body>
{body_html}
</body>
</html>"""

HTML(string=html).write_pdf(pdf_path)
print(f"PDF written to {pdf_path}")
