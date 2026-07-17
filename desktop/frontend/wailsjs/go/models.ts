export namespace apiclient {
	
	export class ModelQuota {
	    label: string;
	    model_id: string;
	    remaining_fraction?: number;
	    remaining_pct?: number;
	    is_exhausted: boolean;
	    reset_time?: string;
	    pool_reset_time?: string;
	    time_until_reset_ms?: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelQuota(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.model_id = source["model_id"];
	        this.remaining_fraction = source["remaining_fraction"];
	        this.remaining_pct = source["remaining_pct"];
	        this.is_exhausted = source["is_exhausted"];
	        this.reset_time = source["reset_time"];
	        this.pool_reset_time = source["pool_reset_time"];
	        this.time_until_reset_ms = source["time_until_reset_ms"];
	    }
	}
	export class Snapshot {
	    email: string;
	    plan_name: string;
	    captured_at: string;
	    staleness_seconds: number;
	    prompt_credits_available: number;
	    prompt_credits_monthly: number;
	    flow_credits_available: number;
	    flow_credits_monthly: number;
	    models: ModelQuota[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.plan_name = source["plan_name"];
	        this.captured_at = source["captured_at"];
	        this.staleness_seconds = source["staleness_seconds"];
	        this.prompt_credits_available = source["prompt_credits_available"];
	        this.prompt_credits_monthly = source["prompt_credits_monthly"];
	        this.flow_credits_available = source["flow_credits_available"];
	        this.flow_credits_monthly = source["flow_credits_monthly"];
	        this.models = this.convertValues(source["models"], ModelQuota);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Account {
	    id: number;
	    email: string;
	    plan_name: string;
	    first_seen: string;
	    last_seen: string;
	    latest_snapshot?: Snapshot;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.email = source["email"];
	        this.plan_name = source["plan_name"];
	        this.first_seen = source["first_seen"];
	        this.last_seen = source["last_seen"];
	        this.latest_snapshot = this.convertValues(source["latest_snapshot"], Snapshot);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AccountRemaining {
	    email: string;
	    remaining_fraction: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountRemaining(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.remaining_fraction = source["remaining_fraction"];
	    }
	}
	export class AccountsResponse {
	    accounts: Account[];
	
	    static createFrom(source: any = {}) {
	        return new AccountsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accounts = this.convertValues(source["accounts"], Account);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BreakdownRow {
	    email: string;
	    label: string;
	    model_id: string;
	    current_fraction?: number;
	    starting_fraction?: number;
	    consumed?: number;
	    reset_time?: string;
	
	    static createFrom(source: any = {}) {
	        return new BreakdownRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.label = source["label"];
	        this.model_id = source["model_id"];
	        this.current_fraction = source["current_fraction"];
	        this.starting_fraction = source["starting_fraction"];
	        this.consumed = source["consumed"];
	        this.reset_time = source["reset_time"];
	    }
	}
	export class BreakdownResponse {
	    rows: BreakdownRow[];
	
	    static createFrom(source: any = {}) {
	        return new BreakdownResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], BreakdownRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CurrentAccount {
	    state: string;
	    is_live: boolean;
	    email?: string;
	    account?: Account;
	    accounts: Account[];
	    last_poll_at?: string;
	    next_poll_at?: string;
	    last_account?: Account;
	    as_of: string;
	
	    static createFrom(source: any = {}) {
	        return new CurrentAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.is_live = source["is_live"];
	        this.email = source["email"];
	        this.account = this.convertValues(source["account"], Account);
	        this.accounts = this.convertValues(source["accounts"], Account);
	        this.last_poll_at = source["last_poll_at"];
	        this.next_poll_at = source["next_poll_at"];
	        this.last_account = this.convertValues(source["last_account"], Account);
	        this.as_of = source["as_of"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DaemonStatus {
	    state: string;
	    emails?: string[];
	    uptime: string;
	    started_at: string;
	    last_poll_at?: string;
	    next_poll_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new DaemonStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.emails = source["emails"];
	        this.uptime = source["uptime"];
	        this.started_at = source["started_at"];
	        this.last_poll_at = source["last_poll_at"];
	        this.next_poll_at = source["next_poll_at"];
	    }
	}
	export class DepletedModel {
	    email: string;
	    label: string;
	    remaining_fraction: number;
	
	    static createFrom(source: any = {}) {
	        return new DepletedModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.label = source["label"];
	        this.remaining_fraction = source["remaining_fraction"];
	    }
	}
	export class Health {
	    status: string;
	    uptime: string;
	
	    static createFrom(source: any = {}) {
	        return new Health(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.uptime = source["uptime"];
	    }
	}
	export class ModelAggregate {
	    label: string;
	    model_id: string;
	    remaining_fraction?: number;
	    remaining_pct?: number;
	    is_exhausted: boolean;
	    reset_time?: string;
	    pool_reset_time?: string;
	    email: string;
	    captured_at: string;
	    staleness_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelAggregate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.model_id = source["model_id"];
	        this.remaining_fraction = source["remaining_fraction"];
	        this.remaining_pct = source["remaining_pct"];
	        this.is_exhausted = source["is_exhausted"];
	        this.reset_time = source["reset_time"];
	        this.pool_reset_time = source["pool_reset_time"];
	        this.email = source["email"];
	        this.captured_at = source["captured_at"];
	        this.staleness_seconds = source["staleness_seconds"];
	    }
	}
	
	export class ModelsLatestResponse {
	    models: ModelAggregate[];
	
	    static createFrom(source: any = {}) {
	        return new ModelsLatestResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.models = this.convertValues(source["models"], ModelAggregate);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NextReset {
	    email: string;
	    label: string;
	    reset_time: string;
	
	    static createFrom(source: any = {}) {
	        return new NextReset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.label = source["label"];
	        this.reset_time = source["reset_time"];
	    }
	}
	
	export class SnapshotsResponse {
	    email: string;
	    snapshots: Snapshot[];
	
	    static createFrom(source: any = {}) {
	        return new SnapshotsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.snapshots = this.convertValues(source["snapshots"], Snapshot);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SparklinePoint {
	    captured_at: string;
	    remaining_fraction?: number;
	
	    static createFrom(source: any = {}) {
	        return new SparklinePoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.captured_at = source["captured_at"];
	        this.remaining_fraction = source["remaining_fraction"];
	    }
	}
	export class SparklineModel {
	    label: string;
	    model_id: string;
	    points: SparklinePoint[];
	
	    static createFrom(source: any = {}) {
	        return new SparklineModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.model_id = source["model_id"];
	        this.points = this.convertValues(source["points"], SparklinePoint);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SparklinesResponse {
	    email: string;
	    models: SparklineModel[];
	
	    static createFrom(source: any = {}) {
	        return new SparklinesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.models = this.convertValues(source["models"], SparklineModel);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Stats {
	    total_polls_this_week: number;
	    most_depleted_model?: DepletedModel;
	    account_most_remaining?: AccountRemaining;
	    next_reset?: NextReset;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_polls_this_week = source["total_polls_this_week"];
	        this.most_depleted_model = this.convertValues(source["most_depleted_model"], DepletedModel);
	        this.account_most_remaining = this.convertValues(source["account_most_remaining"], AccountRemaining);
	        this.next_reset = this.convertValues(source["next_reset"], NextReset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class float64 {
	
	
	    static createFrom(source: any = {}) {
	        return new float64(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class TimelineEvent {
	    type: string;
	    at: string;
	    quota: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new TimelineEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.at = source["at"];
	        this.quota = source["quota"];
	    }
	}
	export class TimelineResponse {
	    email: string;
	    events: TimelineEvent[];
	
	    static createFrom(source: any = {}) {
	        return new TimelineResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.events = this.convertValues(source["events"], TimelineEvent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TimeseriesDay {
	    date: string;
	    providers: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new TimeseriesDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.providers = source["providers"];
	    }
	}
	export class TimeseriesResponse {
	    range: string;
	    agg: string;
	    days: TimeseriesDay[];
	
	    static createFrom(source: any = {}) {
	        return new TimeseriesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = source["range"];
	        this.agg = source["agg"];
	        this.days = this.convertValues(source["days"], TimeseriesDay);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace config {
	
	export class Config {
	    port: number;
	    mask_emails: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.mask_emails = source["mask_emails"];
	    }
	}

}

