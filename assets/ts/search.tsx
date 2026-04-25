interface pageData {
    title: string,
    date: string,
    permalink: string,
    content: string,
    summary?: string,
    tags?: string[],
    categories?: string[],
    image?: string,
    preview: string,
    score: number
}

interface match {
    start: number,
    end: number
}

const tagsToReplace = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
    '…': '&hellip;'
};

function replaceTag(tag: string) {
    return (tagsToReplace as Record<string, string>)[tag] || tag;
}

function replaceHTMLEnt(str: string) {
    return str.replace(/[&<>"']/g, replaceTag);
}

function escapeRegExp(value: string) {
    return value.replace(/[.*+\-?^${}()|[\]\\]/g, '\\$&');
}

function normalizeText(value: string) {
    return (value || '')
        .normalize('NFKC')
        .toLowerCase()
        .replace(/\s+/g, ' ')
        .trim();
}

function tokenizeKeyword(raw: string) {
    const keyword = normalizeText(raw);
    if (!keyword) return [];
    const spaced = keyword.split(/\s+/).filter(Boolean);
    if (spaced.length > 1) return Array.from(new Set(spaced));
    return [keyword];
}

function debounce<T extends (...args: any[]) => void>(fn: T, wait = 120) {
    let timer = 0;
    return (...args: Parameters<T>) => {
        window.clearTimeout(timer);
        timer = window.setTimeout(() => fn(...args), wait);
    };
}

class Search {
    private data: pageData[];
    private form: HTMLFormElement;
    private input: HTMLInputElement;
    private list: HTMLDivElement;
    private resultTitle: HTMLHeadingElement;
    private resultTitleTemplate: string;
    private container: HTMLDivElement;
    private lastSearch = '';

    constructor({ form, input, list, resultTitle, resultTitleTemplate }: {
        form: HTMLFormElement,
        input: HTMLInputElement,
        list: HTMLDivElement,
        resultTitle: HTMLHeadingElement,
        resultTitleTemplate: string
    }) {
        this.form = form;
        this.input = input;
        this.list = list;
        this.resultTitle = resultTitle;
        this.resultTitleTemplate = resultTitleTemplate;
        this.container = list.parentElement as HTMLDivElement;

        if (this.input.value.trim() !== '') {
            void this.doSearch(this.input.value);
        } else {
            this.handleQueryString();
        }

        this.bindQueryStringChange();
        this.bindSearchForm();
    }

    private static processMatches(str: string, matches: match[], ellipsis = true, charLimit = 160, offset = 26): string {
        matches.sort((a, b) => a.start - b.start);

        let i = 0;
        let lastIndex = 0;
        let charCount = 0;
        const resultArray: string[] = [];

        while (i < matches.length) {
            const item = matches[i];

            if (ellipsis && item.start - offset > lastIndex) {
                resultArray.push(`${replaceHTMLEnt(str.substring(lastIndex, lastIndex + offset))} [...] `);
                resultArray.push(replaceHTMLEnt(str.substring(item.start - offset, item.start)));
                charCount += offset * 2;
            } else {
                resultArray.push(replaceHTMLEnt(str.substring(lastIndex, item.start)));
                charCount += item.start - lastIndex;
            }

            let j = i + 1;
            let end = item.end;
            while (j < matches.length && matches[j].start <= end) {
                end = Math.max(matches[j].end, end);
                ++j;
            }

            resultArray.push(`<mark>${replaceHTMLEnt(str.substring(item.start, end))}</mark>`);
            charCount += end - item.start;
            i = j;
            lastIndex = end;

            if (ellipsis && charCount > charLimit) break;
        }

        if (lastIndex < str.length) {
            let end = str.length;
            if (ellipsis) end = Math.min(end, lastIndex + offset);
            resultArray.push(replaceHTMLEnt(str.substring(lastIndex, end)));
            if (ellipsis && end !== str.length) resultArray.push(' [...]');
        }

        return resultArray.join('');
    }

    private static collectMatches(str: string, keywords: string[]): match[] {
        const matches: match[] = [];
        if (!str || keywords.length === 0) return matches;
        const regex = new RegExp(keywords.map(escapeRegExp).join('|'), 'gi');
        for (const item of Array.from(str.matchAll(regex))) {
            matches.push({
                start: item.index || 0,
                end: (item.index || 0) + item[0].length
            });
        }
        return matches;
    }

    private static computeScore(item: pageData, keywords: string[]) {
        const title = normalizeText(item.title);
        const summary = normalizeText(item.summary || '');
        const content = normalizeText(item.content);
        const tags = normalizeText((item.tags || []).join(' '));
        const categories = normalizeText((item.categories || []).join(' '));
        const phrase = normalizeText(keywords.join(' '));

        let score = 0;
        for (const keyword of keywords) {
            if (title.includes(keyword)) score += 12;
            if (tags.includes(keyword) || categories.includes(keyword)) score += 8;
            if (summary.includes(keyword)) score += 5;
            if (content.includes(keyword)) score += 2;
        }

        if (phrase && title.includes(phrase)) score += 18;
        if (phrase && summary.includes(phrase)) score += 9;
        if (phrase && content.includes(phrase)) score += 4;

        return score;
    }

    private async searchKeywords(rawKeyword: string) {
        const rawData = await this.getData();
        const keywords = tokenizeKeyword(rawKeyword);
        if (keywords.length === 0) return [];

        const results: pageData[] = [];
        for (const item of rawData) {
            const score = Search.computeScore(item, keywords);
            if (score <= 0) continue;

            const titleMatches = Search.collectMatches(item.title, keywords);
            const previewSource = item.summary || item.content;
            const previewMatches = Search.collectMatches(previewSource, keywords);

            const result = {
                ...item,
                title: titleMatches.length > 0 ? Search.processMatches(item.title, titleMatches, false) : replaceHTMLEnt(item.title),
                preview: previewMatches.length > 0
                    ? Search.processMatches(previewSource, previewMatches, true)
                    : replaceHTMLEnt((previewSource || '').substring(0, 180)),
                score
            };
            results.push(result);
        }

        return results.sort((a, b) => {
            if (b.score !== a.score) return b.score - a.score;
            return String(b.date || '').localeCompare(String(a.date || ''));
        });
    }

    private async doSearch(rawKeyword: string) {
        const keywords = rawKeyword.trim();
        Search.updateQueryString(keywords, true);

        if (keywords === '') {
            this.lastSearch = '';
            this.clear();
            return;
        }
        if (this.lastSearch === keywords) return;
        this.lastSearch = keywords;

        const startTime = performance.now();
        const results = await this.searchKeywords(keywords);
        this.clear();

        for (const item of results) {
            this.list.append(Search.render(item));
        }

        const endTime = performance.now();
        this.resultTitle.innerText = this.generateResultTitle(results.length, ((endTime - startTime) / 1000).toPrecision(1));
        this.container?.classList.remove('hidden');
    }

    private generateResultTitle(resultLen: number, time: string) {
        return this.resultTitleTemplate.replace('#PAGES_COUNT', resultLen.toString()).replace('#TIME_SECONDS', time);
    }

    public async getData() {
        if (!this.data) {
            const jsonURL = this.form.dataset.json as string;
            this.data = await fetch(jsonURL).then(res => res.json());
            const parser = new DOMParser();

            for (const item of this.data) {
                item.content = parser.parseFromString(item.content || '', 'text/html').body.innerText;
                item.summary = parser.parseFromString(item.summary || '', 'text/html').body.innerText;
                item.preview = '';
                item.score = 0;
            }
        }

        return this.data;
    }

    private bindSearchForm() {
        const runSearch = debounce(() => {
            void this.doSearch(this.input.value);
        }, 140);

        const eventHandler = (e: Event) => {
            e.preventDefault();
            runSearch();
        };

        this.form.addEventListener('submit', eventHandler);
        this.input.addEventListener('input', eventHandler);
        this.input.addEventListener('compositionend', eventHandler);
    }

    private clear() {
        this.list.innerHTML = '';
        this.resultTitle.innerText = '';
        this.container.classList.add('hidden');
    }

    private bindQueryStringChange() {
        window.addEventListener('popstate', () => {
            this.handleQueryString();
        });
    }

    private handleQueryString() {
        const pageURL = new URL(window.location.toString());
        const keywords = pageURL.searchParams.get('keyword') || '';
        this.input.value = keywords;

        if (keywords) {
            void this.doSearch(keywords);
        } else {
            this.clear();
        }
    }

    private static updateQueryString(keywords: string, replaceState = false) {
        const pageURL = new URL(window.location.toString());
        if (keywords === '') {
            pageURL.searchParams.delete('keyword');
        } else {
            pageURL.searchParams.set('keyword', keywords);
        }

        if (replaceState) {
            window.history.replaceState('', '', pageURL.toString());
        } else {
            window.history.pushState('', '', pageURL.toString());
        }
    }

    public static render(item: pageData) {
        const metaParts = []
        if (item.categories && item.categories.length > 0) metaParts.push(replaceHTMLEnt(item.categories.join(' / ')));
        if (item.tags && item.tags.length > 0) metaParts.push(replaceHTMLEnt(item.tags.slice(0, 4).join(' · ')));

        return (
            <article>
                <a href={item.permalink}>
                    <div class="article-details">
                        <h2 class="article-title" dangerouslySetInnerHTML={{ __html: item.title }}></h2>
                        {metaParts.length > 0 &&
                            <div class="article-meta" dangerouslySetInnerHTML={{ __html: metaParts.join(' <span aria-hidden="true">•</span> ') }}></div>
                        }
                        <section class="article-preview" dangerouslySetInnerHTML={{ __html: item.preview }}></section>
                    </div>
                    {item.image &&
                        <div class="article-image">
                            <img src={item.image} loading="lazy" />
                        </div>
                    }
                </a>
            </article>
        );
    }
}

declare global {
    interface Window {
        searchResultTitleTemplate: string;
    }
}

window.addEventListener('load', () => {
    setTimeout(function () {
        const searchForm = document.querySelector('.search-form') as HTMLFormElement,
            searchInput = searchForm?.querySelector('input') as HTMLInputElement,
            searchResultList = document.querySelector('.search-result--list') as HTMLDivElement,
            searchResultTitle = document.querySelector('.search-result--title') as HTMLHeadingElement;

        if (!searchForm || !searchInput || !searchResultList || !searchResultTitle) return;

        new Search({
            form: searchForm,
            input: searchInput,
            list: searchResultList,
            resultTitle: searchResultTitle,
            resultTitleTemplate: window.searchResultTitleTemplate
        });
    }, 0);
});

export default Search;
