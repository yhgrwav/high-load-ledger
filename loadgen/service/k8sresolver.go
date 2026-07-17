package service

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/resolver"
)

// PeriodicDNSScheme is a custom gRPC resolver scheme that re-resolves the target host on a
// fixed timer instead of grpc-go's built-in "dns" resolver, which only re-resolves on an
// explicit ResolveNow() — internally triggered on connection errors, never on a schedule.
// Behind a k8s headless Service, that means new pod IPs added by an HPA scale-up are only
// picked up once an existing backend fails, not proactively. This resolver polls DNS on
// PeriodicDNSInterval regardless, so round_robin load balancing actually rebalances as
// replicas come and go.
const PeriodicDNSScheme = "k8sdns"

// PeriodicDNSInterval is how often the resolver re-looks-up the target host.
var PeriodicDNSInterval = 10 * time.Second

func init() {
	resolver.Register(&periodicDNSBuilder{})
}

type periodicDNSBuilder struct{}

func (b *periodicDNSBuilder) Scheme() string { return PeriodicDNSScheme }

func (b *periodicDNSBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	host, port, err := net.SplitHostPort(target.Endpoint())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &periodicDNSResolver{host: host, port: port, cc: cc, cancel: cancel}
	r.resolve()
	r.wg.Add(1)
	go r.watch(ctx)
	return r, nil
}

type periodicDNSResolver struct {
	host, port string
	cc         resolver.ClientConn
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func (r *periodicDNSResolver) watch(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(PeriodicDNSInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.resolve()
		}
	}
}

func (r *periodicDNSResolver) resolve() {
	ips, err := net.LookupHost(r.host)
	if err != nil {
		r.cc.ReportError(err)
		return
	}
	addrs := make([]resolver.Address, len(ips))
	for i, ip := range ips {
		addrs[i] = resolver.Address{Addr: net.JoinHostPort(ip, r.port)}
	}
	_ = r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *periodicDNSResolver) ResolveNow(resolver.ResolveNowOptions) { r.resolve() }

func (r *periodicDNSResolver) Close() {
	r.cancel()
	r.wg.Wait()
}
