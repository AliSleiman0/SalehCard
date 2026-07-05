import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/review.dart';

/// Product reviews, backed by the API (`/api/v1/reviews`).
abstract interface class ReviewRepository {
  /// Submits a review (stored pending moderation). A repeat review of the same
  /// product surfaces as a [ServerFailure] with code `ALREADY_REVIEWED`.
  Future<Either<Failure, Unit>> submit(ReviewDraft draft);

  /// Whether the caller has already reviewed [productId] (and that review's
  /// moderation status).
  Future<Either<Failure, MyReview>> myReview(String productId);
}
