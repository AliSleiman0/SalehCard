// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'product_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

VariantDto _$VariantDtoFromJson(Map<String, dynamic> json) => VariantDto(
  id: json['id'] as String,
  denomination: json['denomination'] as String? ?? '',
  price: (json['price'] as num?)?.toDouble() ?? 0,
);

Map<String, dynamic> _$VariantDtoToJson(VariantDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'denomination': instance.denomination,
      'price': instance.price,
    };

RatingsDto _$RatingsDtoFromJson(Map<String, dynamic> json) => RatingsDto(
  average: (json['average'] as num?)?.toDouble() ?? 0,
  count: (json['count'] as num?)?.toInt() ?? 0,
);

Map<String, dynamic> _$RatingsDtoToJson(RatingsDto instance) =>
    <String, dynamic>{'average': instance.average, 'count': instance.count};

ProductDto _$ProductDtoFromJson(Map<String, dynamic> json) => ProductDto(
  id: json['id'] as String,
  title: json['title'] as Map<String, dynamic>?,
  category: json['category'] as String?,
  images: (json['images'] as List<dynamic>?)?.map((e) => e as String).toList(),
  variants: (json['variants'] as List<dynamic>?)
      ?.map((e) => VariantDto.fromJson(e as Map<String, dynamic>))
      .toList(),
  stock: (json['stock'] as num?)?.toInt(),
  available: json['available'] as bool?,
  ratings: json['ratings'] == null
      ? null
      : RatingsDto.fromJson(json['ratings'] as Map<String, dynamic>),
);

Map<String, dynamic> _$ProductDtoToJson(ProductDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'category': instance.category,
      'images': instance.images,
      'variants': instance.variants?.map((e) => e.toJson()).toList(),
      'stock': instance.stock,
      'available': instance.available,
      'ratings': instance.ratings?.toJson(),
    };
