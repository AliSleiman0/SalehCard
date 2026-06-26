import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../domain/entities/wallet.dart';
import '../dtos/wallet_dto.dart';

class WalletRemoteDataSource {
  const WalletRemoteDataSource(this._dio);

  final Dio _dio;

  /// GET /wallet → balance + newest-first transactions.
  Future<WalletDto> getWallet() async {
    final response = await _dio.get<dynamic>('/wallet');
    return WalletDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// POST /wallet/topups → the newly-created (already credited) transaction.
  /// Both `card` and `usdt` are mock-approved and credited immediately in dev.
  Future<WalletTxDto> topUp(TopUpInput input) async {
    final response = await _dio.post<dynamic>(
      '/wallet/topups',
      data: {
        'amount': input.amount,
        'method': input.method,
        if (input.ref != null && input.ref!.isNotEmpty) 'ref': input.ref,
      },
    );
    return WalletTxDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }
}
