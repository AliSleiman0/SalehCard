/// The four KYC verification states a customer can be in. Drives which status
/// card the `/kyc` screen renders and the [StatusBadge] colour.
enum KycStatus { unverified, pending, verified, rejected }

/// The document types a customer can submit for verification.
enum KycDocumentType { passport, idCard, license }

/// Wire value for a document type (matches the API's `documentType`).
extension KycDocumentTypeWire on KycDocumentType {
  String get apiValue => switch (this) {
        KycDocumentType.passport => 'passport',
        KycDocumentType.idCard => 'id_card',
        KycDocumentType.license => 'license',
      };
}

/// Parses the API's `status` string into a [KycStatus] (defaults to unverified).
KycStatus kycStatusFromWire(String? value) => switch (value) {
      'pending' => KycStatus.pending,
      'verified' => KycStatus.verified,
      'approved' => KycStatus.verified,
      'rejected' => KycStatus.rejected,
      _ => KycStatus.unverified,
    };

/// The form payload submitted for verification — personal background info only
/// (no document upload).
class KycSubmission {
  const KycSubmission({
    required this.fullName,
    required this.dateOfBirth,
    required this.placeOfBirth,
    required this.placeOfResidence,
    required this.documentType,
    required this.documentNumber,
  });

  final String fullName;

  /// ISO `yyyy-mm-dd`.
  final String dateOfBirth;
  final String placeOfBirth;
  final String placeOfResidence;
  final KycDocumentType documentType;
  final String documentNumber;
}

/// The customer's derived KYC profile. [rejectionReason] is present only when
/// [status] is [KycStatus.rejected].
class KycProfile {
  const KycProfile({required this.status, this.rejectionReason});

  final KycStatus status;
  final String? rejectionReason;
}
