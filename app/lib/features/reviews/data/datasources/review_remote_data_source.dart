import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../domain/entities/review.dart';

/// Talks to the review API (`/api/v1/reviews`). Payloads are small and flat, so
/// they are hand-built/parsed (mirrors the KYC data source — no codegen).
class ReviewRemoteDataSource {
  const ReviewRemoteDataSource(this._dio);

  final Dio _dio;

  /// POST /reviews — submit a review (stored pending moderation). The created
  /// review is discarded; a non-2xx (e.g. 409 already-reviewed) raises a
  /// DioException the repository maps to a [Failure].
  Future<void> submit(ReviewDraft draft) async {
    final response = await _dio.post<dynamic>('/reviews', data: {
      'productId': draft.productId,
      'rating': draft.rating,
      'body': draft.body,
    });
    // Validates the {success, data} envelope (throws on success:false).
    unwrap(response);
  }

  /// GET /reviews/mine?productId= — whether the caller already reviewed a
  /// product, and that review's status.
  Future<MyReview> myReview(String productId) async {
    final response = await _dio.get<dynamic>(
      '/reviews/mine',
      queryParameters: {'productId': productId},
    );
    final data = unwrap(response);
    if (data is! Map) return MyReview.none;
    return MyReview(
      reviewed: data['reviewed'] == true,
      status: data['status'] as String?,
    );
  }
}
