import '../../../../core/i18n/i18n_string.dart';

/// A catalog category (domain entity). Root domains (depth 0) carry a
/// [productCount] when the listing is requested with counts; deeper categories
/// leave it null.
class Category {
  const Category({
    required this.id,
    required this.slug,
    required this.name,
    this.image,
    this.rootDomain,
    this.parentId,
    this.depth = 0,
    this.hasChildren = false,
    this.productCount,
  });

  final String id;
  final String slug;
  final I18nString name;
  final String? image;
  final String? rootDomain;

  /// Parent node id (null for a top-level Collection).
  final String? parentId;
  final int depth;

  /// Whether this node has child categories — the browse UI drills into a
  /// subcategory list when true, and lists products directly when false (leaf).
  final bool hasChildren;
  final int? productCount;
}
