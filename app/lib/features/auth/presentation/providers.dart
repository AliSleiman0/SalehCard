import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/providers.dart';
import '../data/datasources/auth_remote_data_source.dart';
import '../data/repositories/auth_repository_impl.dart';
import '../domain/repositories/auth_repository.dart';
import '../domain/usecases/login.dart';
import '../domain/usecases/login_by_phone.dart';
import '../domain/usecases/request_otp.dart';
import '../domain/usecases/verify_otp.dart';

final authRemoteDataSourceProvider = Provider<AuthRemoteDataSource>(
  (ref) => AuthRemoteDataSource(ref.watch(dioProvider)),
);

final authRepositoryProvider = Provider<AuthRepository>(
  (ref) => AuthRepositoryImpl(
    ref.watch(authRemoteDataSourceProvider),
    ref.watch(tokenStoreProvider),
  ),
);

final loginUseCaseProvider = Provider<Login>(
  (ref) => Login(ref.watch(authRepositoryProvider)),
);

final loginByPhoneUseCaseProvider = Provider<LoginByPhone>(
  (ref) => LoginByPhone(ref.watch(authRepositoryProvider)),
);

final requestOtpUseCaseProvider = Provider<RequestOtp>(
  (ref) => RequestOtp(ref.watch(authRepositoryProvider)),
);

final verifyOtpUseCaseProvider = Provider<VerifyOtp>(
  (ref) => VerifyOtp(ref.watch(authRepositoryProvider)),
);
