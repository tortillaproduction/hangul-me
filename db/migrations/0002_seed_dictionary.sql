-- 開発用シードデータ
-- initial_consonant : 先頭音節の初声（互換字母 ㄱ/ㄴ/ㄷ… で保持）
-- final_consonant   : 末尾音節のパッチム（無ければ NULL）
-- いずれも domain/hangul の Analyze() と同じ規則で導出した値。

BEGIN;

INSERT INTO dictionary_entries
    (hangul, reading_kana, reading_hiragana, romanized, meaning_ja, source,
     initial_consonant, has_final_consonant, final_consonant)
VALUES
    ('안녕',      'アンニョン',       'あんにょん',       'annyeong',      'やあ / こんにちは（友達同士の挨拶）', 'curated', 'ㅇ', true,  'ㅇ'),
    ('안녕하세요','アンニョンハセヨ', 'あんにょんはせよ', 'annyeonghaseyo','こんにちは（丁寧な挨拶）',            'curated', 'ㅇ', false, NULL),
    ('진짜',      'チンチャ',         'ちんちゃ',         'jinjja',        '本当に / マジで',                     'curated', 'ㅈ', false, NULL),
    ('사랑해',    'サランヘ',         'さらんへ',         'saranghae',     '愛してる',                            'curated', 'ㅅ', false, NULL),
    ('오빠',      'オッパ',           'おっぱ',           'oppa',          'お兄さん（女性から年上の男性へ）',    'curated', 'ㅇ', false, NULL),
    ('언니',      'オンニ',           'おんに',           'eonni',         'お姉さん（女性から年上の女性へ）',    'curated', 'ㅇ', false, NULL),
    ('감사합니다','カムサハムニダ',   'かむさはむにだ',   'gamsahamnida',  'ありがとうございます',                'curated', 'ㄱ', false, NULL),
    ('맛있어요',  'マシッソヨ',       'ましっそよ',       'masisseoyo',    'おいしいです',                        'curated', 'ㅁ', false, NULL),
    ('대박',      'テバク',           'てばく',           'daebak',        'すごい / やばい',                     'curated', 'ㄷ', true,  'ㄱ'),
    ('괜찮아요',  'ケンチャナヨ',     'けんちゃなよ',     'gwaenchanayo',  '大丈夫です',                          'curated', 'ㄱ', false, NULL),
    ('보고싶어',  'ポゴシポ',         'ぽごしぽ',         'bogosipeo',     '会いたい',                            'curated', 'ㅂ', false, NULL),
    ('알겠습니다','アルゲッスムニダ', 'あるげっすむにだ', 'algesseumnida', '承知しました',                        'curated', 'ㅇ', false, NULL),
    ('미안해',    'ミアネ',           'みあね',           'mianhae',       'ごめん',                              'curated', 'ㅁ', false, NULL),
    ('화이팅',    'ファイティン',     'ふぁいてぃん',     'hwaiting',      'ファイト / 頑張れ',                   'curated', 'ㅎ', true,  'ㅇ'),
    ('어떻게',    'オットケ',         'おっとけ',         'eotteoke',      'どうしよう / どうやって',             'curated', 'ㅇ', false, NULL)
ON CONFLICT (hangul, meaning_ja) DO NOTHING;

-- 例文（代表的なものだけ）
INSERT INTO example_sentences (entry_id, sentence_ko, sentence_ja, source_title)
SELECT id, '안녕, 오랜만이야.', 'やあ、久しぶり。', NULL
FROM dictionary_entries WHERE hangul = '안녕';

INSERT INTO example_sentences (entry_id, sentence_ko, sentence_ja, source_title)
SELECT id, '진짜 맛있어요!', '本当においしいです！', NULL
FROM dictionary_entries WHERE hangul = '진짜';

INSERT INTO example_sentences (entry_id, sentence_ko, sentence_ja, source_title)
SELECT id, '오빠, 어디 가요?', 'オッパ、どこ行くの？', NULL
FROM dictionary_entries WHERE hangul = '오빠';

INSERT INTO example_sentences (entry_id, sentence_ko, sentence_ja, source_title)
SELECT id, '정말 대박이다!', '本当にすごい！', NULL
FROM dictionary_entries WHERE hangul = '대박';

COMMIT;
