#!/usr/bin/env bash
# Legt die Schemata für `task test:ebilanz-xsd` an.
#
# Lädt die XBRL-Pakete der E-Bilanz-Taxonomie 6.9 und 6.10 und die
# Basisschemata von xbrl.org, prüft jede Datei gegen ihre SHA-256 und legt
# daneben an:
#   6.9.xsd, 6.10.xsd  Wrapper-Schemata, die Instanz, de-gaap-ci, de-gcd und
#                      ab 6.10 de-bra und xbrldi importieren
#   catalog.xml        XML-Katalog, der http://www.xbrl.org/ auf die lokale
#                      Kopie umlenkt; geprüft wird ohne Netz
#
# Die Pakete kommen von xbrl.de (XBRL Deutschland e.V.). Ihr Ordner xbrl/ ist
# byte-identisch mit den XBRL-Paketen auf esteuer.de; esteuer.de sendet ein
# falsches Zwischenzertifikat, an dem curl scheitert. Vorhandene Downloads mit
# richtiger Prüfsumme werden nicht erneut geladen.
#
# Aufruf: scripts/ebilanz_schemas.sh <ordner>
set -euo pipefail

target=${1:?Aufruf: scripts/ebilanz_schemas.sh <ordner>}
mkdir -p "$target/download"
target=$(cd "$target" && pwd)
download=$target/download

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

# fetch <url> <datei unter download/> <sha256>
fetch() {
  local url=$1 file=$download/$2 want=$3 got
  if [ -f "$file" ] && [ "$(sha256 "$file")" = "$want" ]; then
    return
  fi
  mkdir -p "$(dirname "$file")"
  curl -fsSL --retry 3 -o "$file.part" "$url"
  got=$(sha256 "$file.part")
  if [ "$got" != "$want" ]; then
    rm -f "$file.part"
    echo "Prüfsumme stimmt nicht: $url" >&2
    echo "  erhalten $got" >&2
    echo "  erwartet $want" >&2
    exit 1
  fi
  mv "$file.part" "$file"
}

fetch https://www.xbrl.de/german-gaap-taxonomy-v6.9-2025-04-01.zip \
  german-gaap-taxonomy-v6.9-2025-04-01.zip \
  4c2572b438e7c1b04119d2c6363bc9b07605ddb1a14731dec10a3a0900d7bfc9
fetch https://www.xbrl.de/german-gaap-taxonomy-v6.10-2026-04-01.zip \
  german-gaap-taxonomy-v6.10-2026-04-01.zip \
  8e5ceb61a65aa46c6b1b6d86dce2a207ebfae1fecdbb4012687ac2316b258457

while read -r path sum; do
  fetch "https://www.xbrl.org/$path" "xbrl.org/$path" "$sum"
done <<'EOF'
2003/xbrl-instance-2003-12-31.xsd 5dbad94cecf42918b976449a8e1cb800d57e96442a9d0f5c00ca77e4da831827
2003/xbrl-linkbase-2003-12-31.xsd f906cd30bc47a81bd1c97e24447f4917b21eb3d8a8b4fc520129ce2534ac5bca
2003/xl-2003-12-31.xsd e28b0965d34f4879833b433a62d0d1e86e8b1f22e6b7642bab861983f5bb5818
2003/xlink-2003-12-31.xsd 5ffa8cb747ccd11e0d6420cf5668b60e3b7a64e20915d0e1774e5559d3343971
2005/xbrldt-2005.xsd caf850f08d74c087c7946e6ea6d9fb5c0e9f8edf63bf639ccb9143e77e3390c7
2006/xbrldi-2006.xsd eec226ee84a0a1cf61f377780767267ef11f3c8e1a26efe5266a2b0abdc6af05
2006/ref-2006-02-27.xsd fae0144a41edf741c3b74c866988f64d24714dba4ca822258dea8b490b7904b7
EOF

# Nur der Ordner xbrl/ wird gebraucht; der Rest der Pakete ist Begleitmaterial.
for package in german-gaap-taxonomy-v6.9-2025-04-01 german-gaap-taxonomy-v6.10-2026-04-01; do
  rm -rf "${target:?}/$package"
  unzip -q "$download/$package.zip" "$package/xbrl/*" -d "$target"
done

# wrapper <datei> <paketordner> <stichtag> <modul>...
# Importiert die Instanz von xbrl.org und je Modul dessen Schema; xbrldi kommt
# von xbrl.org, alle anderen Module aus dem Paket.
wrapper() {
  local file=$1 package=$2 date=$3 module
  shift 3
  {
    echo '<?xml version="1.0" encoding="UTF-8"?>'
    echo '<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:buchfink:ebilanz-xsd">'
    echo '  <xs:import namespace="http://www.xbrl.org/2003/instance" schemaLocation="http://www.xbrl.org/2003/xbrl-instance-2003-12-31.xsd"/>'
    for module in "$@"; do
      if [ "$module" = xbrldi ]; then
        echo '  <xs:import namespace="http://xbrl.org/2006/xbrldi" schemaLocation="http://www.xbrl.org/2006/xbrldi-2006.xsd"/>'
      else
        echo "  <xs:import namespace=\"http://www.xbrl.de/taxonomies/$module-$date\" schemaLocation=\"$package/xbrl/$module-$date/$module-$date.xsd\"/>"
      fi
    done
    echo '</xs:schema>'
  } >"$target/$file"
}
wrapper 6.9.xsd german-gaap-taxonomy-v6.9-2025-04-01 2025-04-01 de-gaap-ci de-gcd
wrapper 6.10.xsd german-gaap-taxonomy-v6.10-2026-04-01 2026-04-01 de-gaap-ci de-gcd de-bra xbrldi

cat >"$target/catalog.xml" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <rewriteURI uriStartString="http://www.xbrl.org/" rewritePrefix="download/xbrl.org/"/>
  <rewriteSystem systemIdStartString="http://www.xbrl.org/" rewritePrefix="download/xbrl.org/"/>
</catalog>
EOF
