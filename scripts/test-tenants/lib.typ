// Layout-Bausteine für die Testdokumente. Die Daten kommen als JSON aus gen.py.

#let seg(s) = if type(s) == str { s } else if s.at("b", default: false) { strong(s.t) } else if s.at("i", default: false) { emph(s.t) } else { s.t }
#let rich(x) = if type(x) == str { x } else if type(x) == array { x.map(seg).join() } else { seg(x) }
#let lines(xs) = xs.map(rich).join(linebreak())

#let colw(w) = if w == "auto" { auto } else if type(w) == str and w.ends-with("fr") { float(w.slice(0, -2)) * 1fr } else { w * 1mm }
#let colal(a) = if a == "r" { right } else if a == "c" { center } else { left }

#let letterhead(b, acc) = grid(
  columns: (1fr, auto), column-gutter: 8mm,
  {
    text(size: b.at("size", default: 18) * 1pt, weight: "bold", fill: acc, b.name)
    if "tagline" in b { linebreak(); text(size: 8.5pt, fill: luma(90), b.tagline) }
  },
  align(right, text(size: 8.5pt, lines(b.at("right", default: ())))),
)

#let meta(rows) = table(
  columns: (auto, auto), stroke: none, inset: (x: 0pt, y: 1.8pt), column-gutter: 4mm,
  ..rows.map(r => (text(fill: luma(80), r.at(0)), text(weight: "medium", r.at(1)))).flatten(),
)

#let address(b) = grid(
  columns: (92mm, 1fr), column-gutter: 4mm,
  block(height: b.at("height", default: 40) * 1mm, {
    if "sender_line" in b { text(size: 7pt, fill: luma(100), underline(b.sender_line)); v(2mm) }
    text(size: 10pt, lines(b.lines))
  }),
  align(right, text(size: 9pt, meta(b.at("meta", default: ())))),
)

#let datatable(b, acc) = {
  let al = b.align.map(colal)
  let size = b.at("size", default: 9) * 1pt
  table(
    columns: b.widths.map(colw),
    align: (x, y) => al.at(x) + horizon,
    stroke: (x, y) => if y == 0 { (bottom: 0.8pt + acc) } else { (bottom: 0.3pt + luma(205)) },
    inset: (x: 4pt, y: 4pt),
    table.header(..b.header.map(h => text(weight: "bold", size: size - 0.5pt, h))),
    ..b.rows.map(r => r.map(c => text(size: size, rich(c)))).flatten(),
  )
}

#let totals(b, acc) = align(right, table(
  columns: (auto, 34mm), stroke: none, align: (left, right), inset: (x: 4pt, y: 2.6pt),
  ..b.rows.map(r => {
    let bold = r.len() > 2 and r.at(2)
    let w = if bold { "bold" } else { "regular" }
    (text(weight: w, r.at(0)), text(weight: w, r.at(1)))
  }).flatten(),
))

#let kv(b) = table(
  columns: (b.at("label_width", default: 55) * 1mm, 1fr), stroke: (x, y) => (bottom: 0.4pt + luma(170)),
  inset: (x: 3pt, y: 5pt),
  ..b.rows.map(r => (text(fill: luma(70), size: 9pt, r.at(0)), text(size: 10pt, rich(r.at(1))))).flatten(),
)

#let signatures(b) = grid(
  columns: b.items.len() * (1fr,), column-gutter: 12mm,
  ..b.items.map(s => {
    v(4mm)
    box(height: 10mm, align(bottom, text(font: ("Snell Roundhand", "Bradley Hand", "Apple Chancery"), size: 15pt, fill: rgb("#1d3d8f"), s.at("sig", default: ""))))
    line(length: 100%, stroke: 0.5pt)
    text(size: 8pt, fill: luma(80), s.label)
  }),
)

#let notebox(b, acc) = block(
  width: 100%, inset: 8pt, radius: 2pt,
  fill: rgb(b.at("fill", default: "#f3f3f0")), stroke: 0.5pt + acc.lighten(40%),
  text(size: 9pt, lines(b.lines)),
)

#let render-block(b, acc) = {
  let t = b.type
  if t == "letterhead" { letterhead(b, acc); v(8mm) }
  else if t == "address" { address(b); v(4mm) }
  else if t == "title" { text(size: b.at("size", default: 15) * 1pt, weight: "bold", b.text); v(1mm); if "sub" in b { text(size: 9.5pt, fill: luma(70), b.sub) }; v(3mm) }
  else if t == "h" { v(2mm); text(size: 10.5pt, weight: "bold", fill: acc, b.text); v(1mm) }
  else if t == "para" { par(justify: b.at("justify", default: false), text(size: b.at("size", default: 9.5) * 1pt, rich(b.text))); v(1.5mm) }
  else if t == "lines" { text(size: b.at("size", default: 9.5) * 1pt, lines(b.lines)); v(1.5mm) }
  else if t == "table" { datatable(b, acc); v(2mm) }
  else if t == "totals" { totals(b, acc); v(3mm) }
  else if t == "kv" { kv(b); v(3mm) }
  else if t == "sig" { signatures(b); v(3mm) }
  else if t == "note" { notebox(b, acc); v(3mm) }
  else if t == "spacer" { v(b.mm * 1mm) }
  else if t == "pagebreak" { pagebreak() }
}

#let footer(cols, marker, font) = {
  set text(font: font, size: 6.8pt, fill: luma(90))
  line(length: 100%, stroke: 0.4pt + luma(180))
  v(-1mm)
  grid(columns: cols.len() * (1fr,), column-gutter: 4mm, ..cols.map(c => lines(c)))
  v(-1mm)
  align(center, text(size: 6pt, fill: luma(150), marker))
}

#let stamp(s) = place(top + right, dx: -6mm, dy: 52mm, rotate(-10deg, box(
  stroke: 2pt + rgb("#c0392b"), inset: 7pt, radius: 3pt,
  text(size: 22pt, weight: "bold", fill: rgb("#c0392b"), s),
)))

#let a4page(p, marker) = {
  let acc = rgb(p.at("accent", default: "#333333"))
  let foot = p.at("footer", default: ())
  page(
    paper: "a4",
    margin: (top: 16mm, bottom: if foot.len() > 0 { 32mm } else { 18mm }, x: 20mm),
    footer: if foot.len() > 0 { footer(foot, marker, p.at("font", default: "Helvetica")) } else { align(center, text(font: p.at("font", default: "Helvetica"), size: 6pt, fill: luma(150), marker)) },
    {
      set text(font: p.at("font", default: "Helvetica"), size: 9.5pt, lang: p.at("lang", default: "de"))
      if "stamp" in p { stamp(p.stamp) }
      for b in p.blocks { render-block(b, acc) }
    },
  )
}

// Kassenbon: schmal, Festbreitenschrift, Papierton.
#let bonline(l) = {
  if type(l) == str {
    if l == "---" { line(length: 100%, stroke: (dash: "dashed", thickness: 0.5pt)) } else { l }
  } else if type(l) == array {
    grid(columns: (1fr, auto), column-gutter: 2mm, rich(l.at(0)), rich(l.at(1)))
  } else {
    let al = colal(l.at("a", default: "l"))
    align(al, text(size: l.at("s", default: 8.5) * 1pt, weight: if l.at("b", default: false) { "bold" } else { "regular" }, l.t))
  }
}

#let bonpage(p, marker) = page(
  width: p.at("width", default: 80) * 1mm, height: auto, margin: (x: 5mm, y: 7mm),
  fill: rgb(p.at("paper", default: "#fbfaf5")),
  {
    set text(font: p.at("font", default: "Menlo"), size: 8.5pt, lang: "de")
    set par(leading: 0.55em)
    for l in p.lines { block(above: 0pt, below: 0.45em, width: 100%, bonline(l)) }
    v(3mm)
    align(center, text(size: 5.5pt, fill: luma(150), marker))
  },
)

#let render(d) = {
  set document(title: d.title, author: d.author)
  for p in d.pages {
    if p.kind == "bon" { bonpage(p, d.marker) } else { a4page(p, d.marker) }
  }
}
