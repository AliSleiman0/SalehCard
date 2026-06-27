import 'package:dio/dio.dart';

import '../../../../core/i18n/i18n_string.dart';
import '../../../../core/network/api_envelope.dart';
import '../../domain/entities/offer.dart';

/// Fetches the live offers from the public storefront endpoint. The response is
/// hand-parsed (no codegen) since the offer payload is small and flat.
class OffersRemoteDataSource {
  const OffersRemoteDataSource(this._dio);

  final Dio _dio;

  Future<List<Offer>> getOffers() async {
    final response = await _dio.get<dynamic>('/offers');
    final data = unwrap(response) as List<dynamic>;
    return data
        .map((e) => _fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Offer _fromJson(Map<String, dynamic> json) {
    final product = (json['product'] as Map<String, dynamic>?) ?? const {};
    final endsRaw = json['endsAt'] as String?;
    return Offer(
      id: json['id']?.toString() ?? '',
      productId: json['productId']?.toString() ?? '',
      productTitle: I18nString.fromJson(product['title']),
      images: (product['images'] as List<dynamic>?)
              ?.map((e) => e.toString())
              .toList() ??
          const [],
      category: product['category']?.toString() ?? '',
      inStock: product['inStock'] as bool? ?? false,
      discountType: json['discountType']?.toString() ?? 'percent',
      discountValue: _toDouble(json['discountValue']),
      originalFromPrice: _toDouble(json['originalFromPrice']),
      offerFromPrice: _toDouble(json['offerFromPrice']),
      endsAt: endsRaw != null ? DateTime.tryParse(endsRaw) : null,
    );
  }

  static double _toDouble(dynamic v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}
