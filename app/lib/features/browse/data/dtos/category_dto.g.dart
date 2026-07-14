// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'category_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CategoryDto _$CategoryDtoFromJson(Map<String, dynamic> json) => CategoryDto(
  id: json['id'] as String,
  legacyId: (json['legacyId'] as num?)?.toInt(),
  parentLegacyId: (json['parentLegacyId'] as num?)?.toInt(),
  parentId: json['parentId'] as String?,
  slug: json['slug'] as String? ?? '',
  name: json['name'] as Map<String, dynamic>?,
  image: json['image'] as String?,
  rootDomain: json['rootDomain'] as String?,
  depth: (json['depth'] as num?)?.toInt() ?? 0,
  hasChildren: json['hasChildren'] as bool? ?? false,
  productCount: (json['productCount'] as num?)?.toInt(),
);

Map<String, dynamic> _$CategoryDtoToJson(CategoryDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'legacyId': instance.legacyId,
      'parentLegacyId': instance.parentLegacyId,
      'parentId': instance.parentId,
      'slug': instance.slug,
      'name': instance.name,
      'image': instance.image,
      'rootDomain': instance.rootDomain,
      'depth': instance.depth,
      'hasChildren': instance.hasChildren,
      'productCount': instance.productCount,
    };
