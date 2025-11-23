import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '10s', target: 10 },
        { duration: '40s', target: 30 },
        { duration: '10s', target: 0 },
    ],
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
    const teamPayload = JSON.stringify({
        team_name: 'load-test-team',
        members: [
            { user_id: 'u1', username: 'LoadAuthor', is_active: true },
            { user_id: 'u2', username: 'LoadReviewer1', is_active: true },
            { user_id: 'u3', username: 'LoadReviewer2', is_active: true },
        ],
    });

    const res = http.post(`${BASE_URL}/team/add`, teamPayload, {
        headers: { 'Content-Type': 'application/json' },
    });

    check(res, {
        'team created (201)': (r) => r.status === 201,
    });

    return { author_id: 'u1' };
}

export default function (data) {
    const prId = `pr-${__VU}-${__ITER}`;
    const body = JSON.stringify({
        pull_request_id: prId,
        pull_request_name: 'Load test PR',
        author_id: data.author_id,
    });

    const res = http.post(`${BASE_URL}/pullRequest/create`, body, {
        headers: { 'Content-Type': 'application/json' },
    });

    check(res, {
        'create PR 201': (r) => r.status === 201,
    });

    if (__ITER % 5 === 0) {
        const statsRes = http.get(`${BASE_URL}/stats`);
        check(statsRes, {
            'stats 200': (r) => r.status === 200,
        });
    }

    sleep(1);
}