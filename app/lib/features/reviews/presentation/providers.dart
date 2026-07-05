import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/providers.dart';
import '../data/datasources/review_remote_data_source.dart';
import '../data/repositories/review_repository_impl.dart';
import '../domain/entities/review.dart';
import '../domain/repositories/review_repository.dart';
import '../domain/usecases/get_my_review.dart';
import '../domain/usecases/submit_review.dart';

// ---- Reviews (wired to /api/v1/reviews) ----

final reviewRemoteDataSourceProvider = Provider<ReviewRemoteDataSource>(
  (ref) => ReviewRemoteDataSource(ref.watch(dioProvider)),
);

final reviewRepositoryProvider = Provider<ReviewRepository>(
  (ref) => ReviewRepositoryImpl(ref.watch(reviewRemoteDataSourceProvider)),
);

final submitReviewUseCaseProvider = Provider<SubmitReview>(
  (ref) => SubmitReview(ref.watch(reviewRepositoryProvider)),
);

final getMyReviewUseCaseProvider = Provider<GetMyReview>(
  (ref) => GetMyReview(ref.watch(reviewRepositoryProvider)),
);

/// Whether the current user has already reviewed a product — drives the CTA on
/// the product page. Defaults to "not reviewed" on any fetch error so a failed
/// lookup never hides the CTA.
final myReviewProvider =
    FutureProvider.autoDispose.family<MyReview, String>((ref, productId) async {
  final result = await ref.watch(getMyReviewUseCaseProvider).call(productId);
  return result.getOrElse((_) => MyReview.none);
});

/// Submit state for the write-review sheet CTA.
class ReviewFormState {
  const ReviewFormState({this.submitting = false, this.failure});

  final bool submitting;
  final Failure? failure;
}

class ReviewFormController extends Notifier<ReviewFormState> {
  @override
  ReviewFormState build() => const ReviewFormState();

  /// Submits a review. Returns `true` on success (and invalidates
  /// [myReviewProvider] so the CTA flips to "reviewed"), or `false` on failure
  /// (surfaced via [state.failure] for the sheet to message).
  Future<bool> submit(ReviewDraft draft) async {
    if (state.submitting) return false;
    state = const ReviewFormState(submitting: true);
    final result = await ref.read(submitReviewUseCaseProvider).call(draft);
    return result.match(
      (failure) {
        state = ReviewFormState(failure: failure);
        return false;
      },
      (_) {
        state = const ReviewFormState();
        ref.invalidate(myReviewProvider(draft.productId));
        return true;
      },
    );
  }

  void reset() => state = const ReviewFormState();
}

final reviewFormControllerProvider =
    NotifierProvider<ReviewFormController, ReviewFormState>(
  ReviewFormController.new,
);
