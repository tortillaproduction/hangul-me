import { useEffect, useState } from 'react';

/**
 * 学習の記録（分析ビュー）。
 *
 * どのグラフも系列は一つなので、凡例は置かず値を直接ラベルしています。
 * 色はブランドのvioletを面ごとに検証した段階で統一しています。
 */
export function AnalyticsPage() {
    const [data, setData] = useState<AnalyticsOverview | null>(null);
    const [loading, setLoading] = useState(true);
    const [showTable, setShowTable] = useState(false);

    useEffect(() => {
        // ここに学習の記録を取得するロジックを記述
    }, []);

    if (loading) {
        return <p className="px-6 py-12 text-center text-sm opacity-50">読み込み中...</p>;
    }

    if (!data) {
        return <p className="px-6 py-12 text-center text-sm opacity-50">集計を取得できませんてした。</p>;
    }

    const initials: BarDatum[] = data.initialConsonants.map((b) => ({
        label: b.label,
        value: b.count,
    }))
    // 「パッチムなし」は音の種類ではなく有無の話なので、
    // 分布のグラフからは外し、上の一文でまとめて示す。
    const finals: BarDatum[] = data.finalConsonants
        .filter((b) => b.consonant != '')
        .map((b) => ({
            label: b.label,
            value: b.count,
        }))
    const noFinalCount  = data.finalConsonants.find((b) => b.consonant == '')?.count ?? 0;
    const withFinalCount = finals.reduce((s, d) => s + d.value, 0);
    const titles: BarDatum[] = data.encounteredTitles.map((b) => ({
        label: b.label,
        value: b.count,
    }))
    const viewed: BarDatum[] = data.mostViewed
        .filter((w) => w.count > 0)
        .map((w) => ({
            label: w.hangul,
            sub: w.readingKana,
            value: w.count,
        }))
    const searched: BarDatum[] = data.mostSearched.map((w) => ({
        label: w.hangul,
        sub: w.readingKana,
        value: w.count,
    }))

    return ()
}

function Stat({ label, value, unit }: { label: string; value: number; unit: string}) {
    return ()
}

function Panel({
    title,
    note,
    className = '',
    children,
}: {
    title: string;
    note?: string;
    className?: string;
    children: React.ReactNode;
}) {
    return ()
}

/** グラフを読めない環境向けの表ビュー */
function TableView({ data }: { data: AnalyticsOverview }) {
    const sections: { title: string; rows: [string, number][]; unit: string }[] = [
        {
        title: '初声の偏り',
        rows: data.initialConsonants.map((b) => [b.label, b.count]),
        unit: '語',
        },
        {
        title: 'パッチムの分布',
        rows: data.finalConsonants.map((b) => [b.label, b.count]),
        unit: '語',
        },
        {
        title: 'どこで聞いた？',
        rows: data.encounteredTitles.map((b) => [b.label, b.count]),
        unit: '語',
        },
        {
        title: 'よく見返している単語',
        rows: data.mostViewed.map((w) => [`${w.hangul}（${w.readingKana}）`, w.count]),
        unit: '回',
        },
        {
        title: 'よく調べた単語',
        rows: data.mostSearched.map((w) => [`${w.hangul}（${w.readingKana}）`, w.count]),
        unit: '回',
        },
    ]

    return ()
}
