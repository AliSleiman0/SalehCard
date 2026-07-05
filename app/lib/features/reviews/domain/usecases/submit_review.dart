import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/review.dart';
import '../repositories/review_repository.dart';

class SubmitReview {
  const SubmitReview(this._repository);

  final ReviewRepository _repository;

  Future<Either<Failure, Unit>> call(ReviewDraft draft) =>
      _repository.submit(draft);
}
