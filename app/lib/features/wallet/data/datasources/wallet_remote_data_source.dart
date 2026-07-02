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

  /// POST /wallet/topups → the newly-created PENDING request. The wallet is
  /// credited only when an admin approves after confirming the payment.
  Future<TopUpRequestDto> topUp(TopUpInput input) async {
    final response = await _dio.post<dynamic>(
      '/wallet/topups',
      data: {
        'amount': input.amount,
        'channel': input.channel,
        if (input.note != null && input.note!.isNotEmpty) 'note': input.note,
      },
    );
    return TopUpRequestDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// GET /wallet/topups → the user's request history, newest first.
  Future<List<TopUpRequestDto>> listTopUps() async {
    final response = await _dio.get<dynamic>('/wallet/topups');
    final list = unwrap(response) as List<dynamic>? ?? const [];
    return [
      for (final item in list)
        TopUpRequestDto.fromJson(item as Map<String, dynamic>),
    ];
  }
}
