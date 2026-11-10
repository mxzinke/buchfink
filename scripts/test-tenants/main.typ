#import "lib.typ": render
#let d = json(sys.inputs.data)
#for a in d.at("attach", default: ()) {
  pdf.attach(a.name, read(a.path, encoding: none), relationship: a.rel, mime-type: a.mime, description: a.desc)
}
#render(d)
