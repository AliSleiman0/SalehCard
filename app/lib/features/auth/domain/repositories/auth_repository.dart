import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/user.dart';

abstract interface class AuthRepository {
  Future<Either<Failure, User>> login({
    required String email,
    required String password,
  });

  /// Phone + password sign-in.
  Future<Either<Failure, User>> loginByPhone({
    required String phone,
    required String password,
  });

  /// Sends an OTP code to [phone].
  Future<Either<Failure, Unit>> requestOtp({required String phone});

  /// Verifies an OTP code; on success authenticates (creating the account on
  /// first sign-in) and optionally sets [password].
  Future<Either<Failure, User>> verifyOtp({
    required String phone,
    required String code,
    String? password,
  });

  Future<void> logout();
}
