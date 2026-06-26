import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../repositories/transfer_repository.dart';

class SendMoney {
  const SendMoney(this._repository);

  final TransferRepository _repository;

  Future<Either<Failure, Unit>> call({
    required String recipient,
    required double amount,
    String? note,
  }) =>
      _repository.sendMoney(recipient: recipient, amount: amount, note: note);
}
