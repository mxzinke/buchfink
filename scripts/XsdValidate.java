import java.io.File;
import java.net.URI;
import java.util.Arrays;
import javax.xml.XMLConstants;
import javax.xml.catalog.CatalogFeatures;
import javax.xml.catalog.CatalogManager;
import javax.xml.transform.stream.StreamSource;
import javax.xml.validation.SchemaFactory;
import javax.xml.validation.Validator;
import org.xml.sax.ErrorHandler;
import org.xml.sax.SAXException;
import org.xml.sax.SAXParseException;

/**
 * Prüft eine XML-Instanz gegen ein XML-Schema, ohne Netz.
 *
 * Aufruf: java scripts/XsdValidate.java <schema.xsd> <instanz.xml>
 *
 * Verweise auf fremde Schemata löst der XML-Katalog aus XML_CATALOG_FILES auf
 * (Pfade durch Leerzeichen getrennt, wie bei xmllint); andere Adressen als
 * Dateien lädt der Validator nicht. Erfolg ist Rückgabewert 0.
 *
 * Dient als EBILANZ_XSD_VALIDATOR (internal/ebilanz/xsd_test.go): gegen das
 * Schema von de-gaap-ci prüft der Xerces des JDK eine Instanz in etwa zwei
 * Sekunden, xmllint war nach 20 Minuten nicht fertig.
 */
class XsdValidate {
    public static void main(String[] args) throws Exception {
        if (args.length != 2) {
            System.err.println("Aufruf: java XsdValidate.java <schema.xsd> <instanz.xml>");
            System.exit(2);
        }
        SchemaFactory factory = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);
        factory.setProperty(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "file");
        String catalogs = System.getenv("XML_CATALOG_FILES");
        if (catalogs != null && !catalogs.isBlank()) {
            URI[] uris = Arrays.stream(catalogs.trim().split("\\s+"))
                    .map(p -> p.startsWith("file:") ? URI.create(p) : new File(p).toURI())
                    .toArray(URI[]::new);
            CatalogFeatures features = CatalogFeatures.builder()
                    .with(CatalogFeatures.Feature.RESOLVE, "continue").build();
            factory.setResourceResolver(CatalogManager.catalogResolver(features, uris));
        }
        int[] errors = {0};
        ErrorHandler handler = new ErrorHandler() {
            public void warning(SAXParseException e) { report("Warnung", e); }
            public void error(SAXParseException e) { errors[0]++; report("Fehler", e); }
            public void fatalError(SAXParseException e) { errors[0]++; report("Fehler", e); }
        };
        factory.setErrorHandler(handler);
        try {
            Validator validator = factory.newSchema(new File(args[0])).newValidator();
            validator.setErrorHandler(handler);
            validator.validate(new StreamSource(new File(args[1])));
        } catch (SAXException e) {
            if (!(e instanceof SAXParseException)) {
                System.out.println("Fehler " + e.getMessage());
            }
            errors[0]++;
        }
        System.exit(errors[0] == 0 ? 0 : 1);
    }

    static void report(String kind, SAXParseException e) {
        System.out.println(kind + " " + e.getSystemId() + ":" + e.getLineNumber() + ": " + e.getMessage());
    }
}
