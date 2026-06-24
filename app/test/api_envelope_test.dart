import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/error/exceptions.dart';
import 'package:salehcard_app/core/network/api_envelope.dart';

Response<dynamic> _response(dynamic body, {int status = 200}) {
  return Response<dynamic>(
    requestOptions: RequestOptions(path: '/x'),
    data: body,
    statusCode: status,
  );
}

void main() {
  group('unwrap', () {
    test('returns data on a success envelope', () {
      final result = unwrap(_response({
        'success': true,
        'data': {'id': '1'},
      }));
      expect(result, {'id': '1'});
    });

    test('returns a list payload on success', () {
      final result = unwrap(_response({
        'success': true,
        'data': [1, 2, 3],
      }));
      expect(result, [1, 2, 3]);
    });

    test('throws ApiException with code/message on an error envelope', () {
      expect(
        () => unwrap(_response({
          'success': false,
          'error': {'code': 'OUT_OF_STOCK', 'message': 'no codes left'},
        })),
        throwsA(
          isA<ApiException>()
              .having((e) => e.code, 'code', 'OUT_OF_STOCK')
              .having((e) => e.message, 'message', 'no codes left'),
        ),
      );
    });

    test('throws ApiException on an unexpected body', () {
      expect(
        () => unwrap(_response('not a map')),
        throwsA(isA<ApiException>()
            .having((e) => e.code, 'code', 'UNEXPECTED_RESPONSE')),
      );
    });
  });
}
