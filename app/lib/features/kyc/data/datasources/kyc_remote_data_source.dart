import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../domain/entities/kyc.dart';

/// Talks to the KYC API (`/api/v1/kyc`). The `{status, rejectionReason}` payload
/// is small and flat, so it is hand-parsed (no codegen).
class KycRemoteDataSource {
  const KycRemoteDataSource(this._dio);

  final Dio _dio;

  Future<KycProfile> getProfile() async {
    final response = await _dio.get<dynamic>('/kyc/me');
    return _fromJson(unwrap(response) as Map<String, dynamic>);
  }

  Future<KycProfile> submit(KycSubmission s) async {
    final response = await _dio.post<dynamic>('/kyc', data: {
      'fullName': s.fullName,
      'dateOfBirth': s.dateOfBirth,
      'placeOfBirth': s.placeOfBirth,
      'placeOfResidence': s.placeOfResidence,
      'documentType': s.documentType.apiValue,
      'documentNumber': s.documentNumber,
    });
    return _fromJson(unwrap(response) as Map<String, dynamic>);
  }

  KycProfile _fromJson(Map<String, dynamic> json) {
    final reason = json['rejectionReason'] as String?;
    return KycProfile(
      status: kycStatusFromWire(json['status'] as String?),
      rejectionReason: (reason != null && reason.isNotEmpty) ? reason : null,
      submission: _submissionFromJson(json['submission']),
    );
  }

  /// Parses the embedded `submission` object the API returns once a submission
  /// exists (carrying the personal details the customer filed). Null otherwise.
  KycSubmissionDetails? _submissionFromJson(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    return KycSubmissionDetails(
      fullName: (raw['fullName'] as String?) ?? '',
      dateOfBirth: (raw['dateOfBirth'] as String?) ?? '',
      placeOfBirth: (raw['placeOfBirth'] as String?) ?? '',
      placeOfResidence: (raw['placeOfResidence'] as String?) ?? '',
      documentType: kycDocumentTypeFromWire(raw['documentType'] as String?),
      documentNumber: (raw['documentNumber'] as String?) ?? '',
    );
  }
}
