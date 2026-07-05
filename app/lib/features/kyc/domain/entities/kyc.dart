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

/// Parses the API's `documentType` string into a [KycDocumentType] (defaults to
/// [KycDocumentType.idCard]).
KycDocumentType kycDocumentTypeFromWire(String? value) => switch (value) {
      'passport' => KycDocumentType.passport,
      'license' => KycDocumentType.license,
      _ => KycDocumentType.idCard,
    };

/// Parses the API's `status` string into a [KycStatus] (defaults to unverified).
KycStatus kycStatusFromWire(String? value) => switch (value) {
      'pending' => KycStatus.pending,
      'verified' => KycStatus.verified,
      'approved' => KycStatus.verified,
      'rejected' => KycStatus.rejected,
      _ => KycStatus.unverified,
    };

/// The form payload submitted for verification — personal background info plus
/// the uploaded document-photo URLs (front always; back required for
/// id_card/license, optional for passports).
class KycSubmission {
  const KycSubmission({
    required this.fullName,
    required this.dateOfBirth,
    required this.placeOfBirth,
    required this.placeOfResidence,
    required this.documentType,
    required this.documentNumber,
    required this.documentFrontUrl,
    this.documentBackUrl,
  });

  final String fullName;

  /// ISO `yyyy-mm-dd`.
  final String dateOfBirth;
  final String placeOfBirth;
  final String placeOfResidence;
  final KycDocumentType documentType;
  final String documentNumber;

  /// Public URL returned by `POST /kyc/documents` for the front photo.
  final String documentFrontUrl;

  /// Back-photo URL; null only when the document is a passport.
  final String? documentBackUrl;
}

/// The stored details of a customer's KYC submission, echoed back by
/// `GET /kyc/me` so the profile can display what was submitted. Null until the
/// customer has filed a submission.
class KycSubmissionDetails {
  const KycSubmissionDetails({
    required this.fullName,
    required this.dateOfBirth,
    required this.placeOfBirth,
    required this.placeOfResidence,
    required this.documentType,
    required this.documentNumber,
    this.documentFrontUrl,
    this.documentBackUrl,
  });

  final String fullName;

  /// ISO `yyyy-mm-dd`.
  final String dateOfBirth;
  final String placeOfBirth;
  final String placeOfResidence;
  final KycDocumentType documentType;
  final String documentNumber;
  final String? documentFrontUrl;
  final String? documentBackUrl;
}

/// The customer's derived KYC profile. [rejectionReason] is present only when
/// [status] is [KycStatus.rejected]; [submission] carries the submitted details
/// once the customer has filed one (used to display them on the profile).
class KycProfile {
  const KycProfile({
    required this.status,
    this.rejectionReason,
    this.submission,
  });

  final KycStatus status;
  final String? rejectionReason;
  final KycSubmissionDetails? submission;
}
