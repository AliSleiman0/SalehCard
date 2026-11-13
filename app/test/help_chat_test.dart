import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/help/data/chatbot_repository.dart';
import 'package:salehcard_app/features/help/domain/chat_logic.dart';

void main() {
  final data = ChatbotData.fromJson(<String, dynamic>{
    'entries': [
      {
        'id': 'top',
        'question': {'en': 'Top up wallet', 'ar': 'شحن'},
        'answer': {'en': 'a'},
        'keywords': ['wallet'],
        'sortOrder': 2,
      },
      {
        'id': 'del',
        'question': {'en': 'Delivery time'},
        'answer': {'en': 'b'},
        'sortOrder': 1,
      },
      {
        'id': 'f1',
        'parentId': 'top',
        'question': {'en': 'Approval'},
        'answer': {'en': 'c'},
      },
    ],
    'support': {'whatsapp': '+961 70 123 456'},
  });

  test('parses entries + support', () {
    expect(data.entries.length, 3);
    expect(data.support.whatsapp, '+961 70 123 456');
    expect(data.support.isEmpty, isFalse);
  });

  test('orders questions and follow-ups', () {
    expect(topLevelQuestions(data.entries).map((e) => e.id), ['del', 'top']);
    expect(followUpQuestions(data.entries, 'top').map((e) => e.id), ['f1']);
  });

  test('searches question text and keywords', () {
    expect(searchQuestions(data.entries, 'deliv').map((e) => e.id), ['del']);
    expect(searchQuestions(data.entries, 'شحن').map((e) => e.id), ['top']);
    expect(searchQuestions(data.entries, 'WALLET').map((e) => e.id), ['top']);
    expect(searchQuestions(data.entries, ' '), isEmpty);
  });

  test('whatsapp link', () {
    expect(
      whatsappUri('+961 70 123 456').toString(),
      'https://wa.me/96170123456',
    );
    expect(whatsappUri('x'), isNull);
  });
}
