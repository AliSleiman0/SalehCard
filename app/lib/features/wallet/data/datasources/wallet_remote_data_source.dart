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
        if (input.channel.isNotEmpty) 'channel': input.channel,
        if (input.methodId != null && input.methodId!.isNotEmpty)
          'methodId': input.methodId,
        if (input.fields.isNotEmpty)
          'fields': [
            for (final e in input.fields.entries)
              {'key': e.key, 'value': e.value},
          ],
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

  /// GET /wallet/topup-methods → the enabled manual funding methods.
  Future<List<TopUpMethodDto>> listMethods() async {
    final response = await _dio.get<dynamic>('/wallet/topup-methods');
    final list = unwrap(response) as List<dynamic>? ?? const [];
    return [
      for (final item in list)
        TopUpMethodDto.fromJson(item as Map<String, dynamic>),
    ];
  }

  /// POST /wallet/topups/documents (multipart `image`) → the stored public URL
  /// the customer submits as a file-field value. Dio sets the multipart
  /// content-type itself for [FormData] bodies.
  Future<String> uploadDocument(String filePath) async {
    final form = FormData.fromMap({
      'image': await MultipartFile.fromFile(filePath, filename: 'proof.jpg'),
    });
    final response = await _dio.post<dynamic>('/wallet/topups/documents', data: form);
    return (unwrap(response) as Map<String, dynamic>)['url'] as String;
  }
}
