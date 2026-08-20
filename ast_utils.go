package eslint

// astUtils helpers transcribed from eslint rules/utils/ast-utils.js
// (getStaticPropertyName / getStaticStringValue) plus the null-literal check.

// isNullLiteral reports whether a Literal with a null value is a null
// literal rather than a regex/bigint (based on the raw source).
func isNullLiteral(node Node) bool {
	return getString(node, "raw") == "null"
}

// getStaticStringValue returns the static string value of a Literal or a
// single-quasi TemplateLiteral, or "" (null) if not statically a string.
func getStaticStringValue(node Node) string {
	switch nodeType(node) {
	case "Literal":
		v, hasValue := node["value"]
		if !hasValue || v == nil {
			raw := getString(node, "raw")
			if raw == "null" {
				return "null"
			}
			if _, ok := node["regex"]; ok {
				re, _ := node["regex"].(map[string]any)
				pat := getString(Node(re), "pattern")
				flags := getString(Node(re), "flags")
				return "/" + pat + "/" + flags
			}
			if b, ok := node["bigint"].(string); ok {
				return b
			}
			return ""
		}
		return valueString(v)
	case "TemplateLiteral":
		exprs := getNodes(node, "expressions")
		quasis := getNodes(node, "quasis")
		if len(exprs) == 0 && len(quasis) == 1 {
			cooked := getNode(quasis[0], "value")
			if cooked != nil {
				if c, ok := cooked["cooked"].(string); ok {
					return c
				}
			}
		}
	}
	return ""
}

// getStaticPropertyName returns the static property name of a node, or ""
// if the name is dynamic.
func getStaticPropertyName(node Node) string {
	var prop Node
	switch nodeType(node) {
	case "ChainExpression":
		return getStaticPropertyName(getNode(node, "expression"))
	case "Property", "PropertyDefinition", "MethodDefinition":
		prop = getNode(node, "key")
	case "MemberExpression":
		prop = getNode(node, "property")
	}
	if prop != nil {
		if nodeType(prop) == "Identifier" && !getBool(node, "computed") {
			return getString(prop, "name")
		}
		return getStaticStringValue(prop)
	}
	return ""
}
