// APIのレスポンス型。backendの interface/http のJSONと対応します。

/** 単語を調べたときの候補1件 */
export interface Candidate {
  /** 辞書DBに実体がある場合のみ非null。nullはAI補完の候補 */
  entryId: string | null;
  hangul: string;
  readingKana: string;
  romanized: string;
  meaningJa: string;
}

export interface LookupResult {
  query: string;
  matchedVia: 'db_trgm' | 'ai_fallback' | 'none';
  candidates: Candidate[];
}

export interface ExampleSentence {
  sentenceKo: string;
  sentenceJa: string;
  sourceTitle?: string;
}

/** 単語帳に登録された単語 */
export interface Word {
  id: string;
  entryId: string;
  hangul: string;
  readingKana: string;
  romanized: string;
  meaningJa: string;
  memo: string;
  encounteredTitle: string;
  masteryLevel: number;
  masteryLabel: string;
  initialConsonant: string;
  hasFinalConsonant: boolean;
  finalConsonant: string;
  registeredAt: string;
  examples: ExampleSentence[];
}

export interface RegisterWordInput {
  entryId?: string | null;
  hangul?: string;
  readingKana?: string;
  readingHiragana?: string;
  romanized?: string;
  meaningJa?: string;
  memo?: string;
  encounteredTitle?: string;
  rawQuery?: string;
  matchedVia?: string;
}

export interface RankedWord {
  entryId: string;
  hangul: string;
  readingKana: string;
  meaningJa: string;
  count: number;
}

export interface ConsonantBucket {
  consonant: string;
  label: string;
  count: number;
}

export interface TitleBucket {
  title: string;
  label: string;
  count: number;
}

export interface DailyCount {
  date: string;
  count: number;
}

export interface AnalyticsOverview {
  totalWords: number;
  registeredThisWeek: number;
  mostSearched: RankedWord[];
  mostViewed: RankedWord[];
  initialConsonants: ConsonantBucket[];
  finalConsonants: ConsonantBucket[];
  encounteredTitles: TitleBucket[];
  registrationTrend: DailyCount[];
}

export interface CurrentUser {
  id: string;
  email: string;
  displayName: string;
  plan: string;
  wordLimit: number;
}
