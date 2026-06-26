import 'package:json_annotation/json_annotation.dart';

import '../../../../core/i18n/i18n_string.dart';
import '../../domain/entities/category.dart';

part 'category_dto.g.dart';

@JsonSerializable()
class CategoryDto {
  const CategoryDto({
    required this.id,
    this.legacyId,
    this.parentLegacyId,
    this.slug = '',
    this.name,
    this.image,
    this.rootDomain,
    this.depth = 0,
    this.productCount,
  });

  final String id;
  final int? legacyId;
  final int? parentLegacyId;
  final String slug;
  final Map<String, dynamic>? name;
  final String? image;
  final String? rootDomain;
  final int depth;
  final int? productCount;

  factory CategoryDto.fromJson(Map<String, dynamic> json) =>
      _$CategoryDtoFromJson(json);

  Map<String, dynamic> toJson() => _$CategoryDtoToJson(this);

  Category toEntity() => Category(
        id: id,
        slug: slug,
        name: I18nString.fromJson(name),
        image: image,
        rootDomain: rootDomain,
        depth: depth,
        productCount: productCount,
      );
}
