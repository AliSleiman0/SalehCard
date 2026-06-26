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
    this.depth = 0,
    this.productCount,
  });

  final String id;
  final String slug;
  final I18nString name;
  final String? image;
  final String? rootDomain;
  final int depth;
  final int? productCount;
}
