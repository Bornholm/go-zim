package zim

type Namespace string

const (
	V6NamespaceContent   Namespace = "C"
	V6NamespaceMetadata  Namespace = "M"
	V6NamespaceWellKnown Namespace = "W"
	V6NamespaceSearch    Namespace = "X"
)

const (
	V5NamespaceLayout              Namespace = "-"
	V5NamespaceArticle             Namespace = "A"
	V5NamespaceArticleMetadata     Namespace = "B"
	V5NamespaceImageFile           Namespace = "I"
	V5NamespaceImageText           Namespace = "J"
	V5NamespaceMetadata            Namespace = "M"
	V5NamespaceCategoryText        Namespace = "U"
	V5NamespaceCategoryArticleList Namespace = "V"
	V5NamespaceCategoryPerArticle  Namespace = "W"
	V5NamespaceSearch              Namespace = "X"
)
